package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	api "github.com/borchero/meerkat-operator/pkg/api/v1alpha1"
	"github.com/borchero/meerkat-operator/pkg/controllers/ovpnserver"
	"github.com/borchero/meerkat-operator/pkg/crypto"
	"github.com/borchero/meerkat-operator/pkg/ovpn"
	vaultapi "github.com/hashicorp/vault/api"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=meerkat.borchero.com,resources=ovpnservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meerkat.borchero.com,resources=ovpnservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// OvpnServerReconciler reconciles OvpnServer objects.
type OvpnServerReconciler struct {
	client.Client
	config Config
	vault  *vaultapi.Client
	scheme *runtime.Scheme
	logger *zap.Logger
}

// MustSetupOvpnServerReconciler initializes a new server reconciler and attaches it to the given
// manager. It panics on failure.
func MustSetupOvpnServerReconciler(
	config Config, vault *vaultapi.Client, mgr ctrl.Manager, logger *zap.Logger,
) {
	reconciler := &OvpnServerReconciler{
		Client: mgr.GetClient(),
		config: config,
		vault:  vault,
		scheme: mgr.GetScheme(),
		logger: logger,
	}
	if err := reconciler.setupWithManager(mgr); err != nil {
		panic(err)
	}
}

func (r *OvpnServerReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.OvpnServer{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

//-------------------------------------------------------------------------------------------------

const (
	secretKeyDh            = "dh.pem"
	secretKeyTa            = "ta.key"
	secretKeyCrl           = "crl.pem"
	secretKeyServerCrt     = "server.crt"
	secretKeyServerKey     = "server.key"
	secretKeyCaCrt         = "ca.crt"
	secretKeySerial        = "serial"
	configMapKeyEntrypoint = "entrypoint.sh"
	configMapKeyOvpnConfig = "openvpn.conf"

	annotationKeyExpiresAt = "meerkat.borchero.com/expires-at"

	finalizerIdentifier = "finalizers.meerkat.borchero.com"
)

// Reconcile reconciles the given request.
func (r *OvpnServerReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	logger := r.logger.With(zap.String("name", req.String()))
	logger.Info("starting reconciliation")

	// First, we get the server - if it cannot be found, we return no error
	server := &api.OvpnServer{}
	err := r.Get(ctx, req.NamespacedName, server)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// The server currently exists, so we check if we need to delete it.
	if !server.DeletionTimestamp.IsZero() {
		// In that case, we need to make sure that we delete the PKI associated with the server
		if controllerutil.ContainsFinalizer(server, finalizerIdentifier) {
			if err := r.deletePKI(ctx, server); err != nil {
				logger.Error("failed purging PKI", zap.Error(err))
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(server, finalizerIdentifier)
			if err := r.Update(ctx, server); err != nil {
				logger.Error("failed removing finalizer", zap.Error(err))
				return ctrl.Result{}, err
			}
		}
		// Returning here tells Kubernetes that the server and its dependents can be removed
		return ctrl.Result{}, nil
	}
	logger.Debug("server has not been deleted")

	// Prior to anything, we add our finalizer if required
	logger.Debug("reconciling finalizers")
	if !controllerutil.ContainsFinalizer(server, finalizerIdentifier) {
		controllerutil.AddFinalizer(server, finalizerIdentifier)
		if err := r.Update(ctx, server); err != nil {
			logger.Error("failed adding finalizer", zap.Error(err))
			return ctrl.Result{}, err
		}
	}

	// Otherwise, the server is not being deleted, so we can reconcile. First, we want to ensure
	// that the shared secrets exist.
	logger.Debug("reconciling shared secrets")
	if err := r.updateSharedSecret(ctx, server, logger); err != nil {
		logger.Error("failed to reconcile shared secrets", zap.Error(err))
		return ctrl.Result{}, err
	}

	// Afterwards, we make sure that the PKI is established correctly.
	logger.Debug("reconciling PKI")
	if err := r.updatePKI(ctx, server, logger); err != nil {
		logger.Error("failed to reconcile PKI", zap.Error(err))
		return ctrl.Result{}, err
	}

	// As soon as that succeeded, we can create a certificate for the server to use. We use the
	// `expiresAt` value to set an annotation on the deployment pods to reload them as soon as
	// a new certificate has been generated.
	logger.Debug("reconciling server certificate")
	expiresAt, err := r.updateServerCertificate(ctx, server, logger)
	if err != nil {
		logger.Error("failed to reconcile server certificate", zap.Error(err))
		return ctrl.Result{}, err
	}

	// We can then update the configuration, entrypoint, deployment, and service
	logger.Debug("reconciling k8s resources")
	if err := r.updateConfigMaps(ctx, server, logger); err != nil {
		logger.Error("failed to reconcile configmaps", zap.Error(err))
		return ctrl.Result{}, err
	}
	if err := r.updateDeployment(ctx, server, expiresAt, logger); err != nil {
		logger.Error("failed to reconcile deployment", zap.Error(err))
		return ctrl.Result{}, err
	}
	if err := r.updateService(ctx, server, logger); err != nil {
		logger.Error("failed to reconcile service", zap.Error(err))
		return ctrl.Result{}, err
	}

	logger.Info("reconciliation succeeded")
	return ctrl.Result{}, nil
}

//-------------------------------------------------------------------------------------------------

func (r *OvpnServerReconciler) deletePKI(ctx context.Context, server *api.OvpnServer) error {
	pki := r.getPKI(server)
	return pki.DisableIfEnabled()
}

//-------------------------------------------------------------------------------------------------

func (r *OvpnServerReconciler) updateSharedSecret(
	ctx context.Context, server *api.OvpnServer, logger *zap.Logger,
) error {
	secret := &corev1.Secret{ObjectMeta: server.ObjectRefSharedSecrets()}

	// If the secret already exists and the keys exist, we don't have to do anything
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for shared secret: %s", err)
	}
	if !apierrors.IsNotFound(err) {
		_, dhExists := secret.Data[secretKeyDh]
		_, taExists := secret.Data[secretKeyTa]
		if dhExists && taExists {
			return nil
		}
	}

	// Otherwise, we need to generate them
	logger.Info("generating DH parameters, this will take a long time")
	bits := server.Spec.Security.DiffieHellmanBits
	if bits == 0 {
		bits = 2048
	}
	dh, err := crypto.GenerateDhParams(bits)
	if err != nil {
		return fmt.Errorf("failed to generate DH params: %s", err)
	}
	ta, err := crypto.GenerateTLSAuth()
	if err != nil {
		return fmt.Errorf("failed to generate TLS auth: %s", err)
	}

	data := map[string][]byte{secretKeyDh: dh, secretKeyTa: ta}
	op, err := ctrl.CreateOrUpdate(ctx, r, secret, func() error {
		secret.Data = data
		return ctrl.SetControllerReference(server, secret, r.scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to upsert shared secret: %s", err)
	}
	logger.Debug("reconciled shared secret", zap.String("operation", string(op)))
	return nil
}

func (r *OvpnServerReconciler) updatePKI(
	ctx context.Context, server *api.OvpnServer, logger *zap.Logger,
) error {
	pki := r.getPKI(server)

	// First, we make sure that everything is configured correctly
	if err := pki.EnsureEnabled(); err != nil {
		return fmt.Errorf("failed to ensure that PKI engine is enabled: %s", err)
	}
	if err := pki.GenerateRootIfRequired(ovpnserver.PKIConfig(server)); err != nil {
		return fmt.Errorf("failed to ensure root certificate: %s", err)
	}
	if err := pki.ConfigureRole("server", ovpnserver.PKIServerConfig(server)); err != nil {
		return fmt.Errorf("failed to ensure server configuration: %s", err)
	}
	if err := pki.ConfigureRole("client", ovpnserver.PKIClientConfig(server)); err != nil {
		return fmt.Errorf("failed to ensure client configuration: %s", err)
	}

	// Then, we pull the CRL into the respective secret
	crl, err := pki.GetCRL()
	if err != nil {
		return fmt.Errorf("failed to fetch up-to-date CRL: %s", err)
	}

	secret := &corev1.Secret{ObjectMeta: server.ObjectRefCrlSecret()}
	op, err := ctrl.CreateOrUpdate(ctx, r, secret, func() error {
		secret.Annotations = map[string]string{}
		secret.Data = map[string][]byte{
			secretKeyCrl: []byte(crl.Certificate),
		}
		return ctrl.SetControllerReference(server, secret, r.scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to update CRL secret: %s", err)
	}
	logger.Debug("updated CRL", zap.String("operation", string(op)))
	return nil
}

func (r *OvpnServerReconciler) updateServerCertificate(
	ctx context.Context, server *api.OvpnServer, logger *zap.Logger,
) (string, error) {
	// First, we get the certificate
	secret := &corev1.Secret{ObjectMeta: server.ObjectRefServerCertificateSecret()}
	if err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return "", fmt.Errorf("failed to get existing secret: %s", err)
		}
	}

	// If it exists, we parse the expiration date and check if it is far in the future (more than
	// one sixth of its validity). If so, we return without error
	if expiresAt, ok := secret.Annotations[annotationKeyExpiresAt]; ok {
		deadline, err := time.Parse(time.RFC3339, expiresAt)
		if err == nil {
			remaining := deadline.Sub(time.Now())
			if remaining > server.Spec.Security.Server.DefaultedValidity()/6 {
				return expiresAt, nil
			}
		}
	}

	// Otherwise, we issue a new certificate...
	pki := r.getPKI(server)
	cert, err := pki.Generate(
		"server", server.Spec.Network.Host, server.Spec.Security.Server.DefaultedValidity(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate new certificate: %s", err)
	}

	// ... and update the secret accordingly
	expiresAt := cert.Expiration.Format(time.RFC3339)
	op, err := ctrl.CreateOrUpdate(ctx, r, secret, func() error {
		secret.Annotations = map[string]string{
			annotationKeyExpiresAt: expiresAt,
		}
		secret.Data = map[string][]byte{
			secretKeyServerCrt: []byte(cert.Certificate),
			secretKeyServerKey: []byte(cert.PrivateKey),
			secretKeyCaCrt:     []byte(cert.CACertificate),
		}
		return ctrl.SetControllerReference(server, secret, r.scheme)
	})
	if err != nil {
		return "", fmt.Errorf("failed to upsert server certificate secret: %s", err)
	}
	logger.Debug("updated server certificate", zap.String("operation", string(op)))
	return expiresAt, nil
}

func (r *OvpnServerReconciler) updateConfigMaps(
	ctx context.Context, server *api.OvpnServer, logger *zap.Logger,
) error {
	// First, let's update the entrypoint
	cm := &corev1.ConfigMap{ObjectMeta: server.ObjectRefEntrypointConfigMap()}
	entrypointValues := ovpn.EntrypointValues{
		Routes: ovpn.ParseRoutesString(server.Spec.Traffic.Routes),
	}
	data, err := ovpn.GetEntrypoint(entrypointValues)
	if err != nil {
		return fmt.Errorf("failed to get code for entrypoint: %s", err)
	}

	op, err := ctrl.CreateOrUpdate(ctx, r, cm, func() error {
		cm.Data = map[string]string{configMapKeyEntrypoint: data}
		return ctrl.SetControllerReference(server, cm, r.scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to upsert server entrypoint: %s", err)
	}
	logger.Debug("updated entrypoint", zap.String("operation", string(op)))

	// Then, update the configuration
	cm = &corev1.ConfigMap{ObjectMeta: server.ObjectRefOvpnConfigMap()}
	configValues := ovpn.ConfigValues{
		Nameservers: server.Spec.Traffic.DefaultedNameservers(),
		RedirectAll: server.Spec.Traffic.RedirectAll,
		Protocol:    string(server.Spec.Network.Protocol),
		Routes:      ovpn.ParseRoutes(server.Spec.Traffic.Routes),
		Security: ovpn.ConfigSecurity{
			Hmac:   string(server.Spec.Security.DefaultedHmac()),
			Cipher: string(server.Spec.Security.DefaultedCipher()),
		},
		Files: ovpn.ConfigFiles{
			TLSServerCrt: filepath.Join(ovpnserver.MountPathTLSKeys, secretKeyServerCrt),
			TLSServerKey: filepath.Join(ovpnserver.MountPathTLSKeys, secretKeyServerKey),
			TLSCaCrt:     filepath.Join(ovpnserver.MountPathTLSKeys, secretKeyCaCrt),
			DHParams:     filepath.Join(ovpnserver.MountPathSharedSecrets, secretKeyDh),
			TLSAuth:      filepath.Join(ovpnserver.MountPathSharedSecrets, secretKeyTa),
			CRL:          filepath.Join(ovpnserver.MountPathCrl, secretKeyCrl),
		},
	}
	data, err = ovpn.GetConfig(configValues)
	if err != nil {
		return fmt.Errorf("failed to get OVPN config: %s", err)
	}

	op, err = ctrl.CreateOrUpdate(ctx, r, cm, func() error {
		cm.Data = map[string]string{configMapKeyOvpnConfig: data}
		return ctrl.SetControllerReference(server, cm, r.scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to upsert server config: %s", err)
	}
	logger.Debug("updated server config", zap.String("operation", string(op)))
	return nil
}

func (r *OvpnServerReconciler) updateDeployment(
	ctx context.Context, server *api.OvpnServer, certificateExpiration string, logger *zap.Logger,
) error {
	deployment := &appsv1.Deployment{ObjectMeta: server.ObjectRefDeployment()}
	expected := ovpnserver.GetDeploymentSpec(server, r.config.Image, map[string]string{
		annotationKeyExpiresAt: certificateExpiration,
	})

	op, err := controllerutil.CreateOrPatch(ctx, r, deployment, func() error {
		if server.Spec.Deployment.Annotations != nil {
			deployment.Annotations = server.Spec.Deployment.Annotations
		}
		deployment.Spec = expected
		return ctrl.SetControllerReference(server, deployment, r.scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to upsert deployment: %s", err)
	}
	logger.Debug("updated deployment", zap.String("operation", string(op)))
	return nil
}

func (r *OvpnServerReconciler) updateService(
	ctx context.Context, server *api.OvpnServer, logger *zap.Logger,
) error {
	service := &corev1.Service{ObjectMeta: server.ObjectRefService()}

	// First, we need to get the service
	err := r.Get(ctx, client.ObjectKeyFromObject(service), service)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to fetch current service: %s", err)
	}
	expected := ovpnserver.GetServiceSpec(server)

	// If it exists, we need to check the type
	if !apierrors.IsNotFound(err) {
		if service.Spec.Type != server.Spec.Service.DefaultedServiceType() {
			// If the type is not equal, we need to delete the service
			if err := r.Delete(ctx, service); err != nil {
				return fmt.Errorf("failed to delete out-of-date service: %s", err)
			}
			logger.Debug("deleted old service")
		} else {
			// Otherwise, we potentially update if there are changes
			updated := false
			if !equality.Semantic.DeepEqual(service.Annotations, server.Spec.Service.Annotations) {
				service.Annotations = server.Spec.Service.Annotations
				updated = true
			}
			if !equality.Semantic.DeepEqual(service.Spec.Selector, expected.Selector) {
				service.Spec.Selector = expected.Selector
				updated = true
			}
			if expected.Type == corev1.ServiceTypeLoadBalancer && len(service.Spec.Ports) > 0 {
				// We need to make sure that the expected node port is not 0
				expected.Ports[0].NodePort = service.Spec.Ports[0].NodePort
			}
			if !equality.Semantic.DeepEqual(service.Spec.Ports, expected.Ports) {
				service.Spec.Ports = expected.Ports
				updated = true
			}
			if updated {
				if err := r.Update(ctx, service); err != nil {
					return fmt.Errorf("failed to update out-of-date service: %s", err)
				}
				logger.Debug("updated outdated service")
			}
			return nil
		}
	}

	// If it doesn't exist, we create it
	service.Annotations = server.Spec.Service.Annotations
	expected.DeepCopyInto(&service.Spec)
	if err := ctrl.SetControllerReference(server, service, r.scheme); err != nil {
		return fmt.Errorf("failed to set owner of service: %s", err)
	}
	if err := r.Create(ctx, service); err != nil {
		return fmt.Errorf("failed to create service: %s", err)
	}
	logger.Debug("created new service")
	return nil
}

//-------------------------------------------------------------------------------------------------

func (r *OvpnServerReconciler) getPKI(server *api.OvpnServer) *crypto.PKI {
	return crypto.NewPKI(
		r.vault, fmt.Sprintf("%s/%s/%s", r.config.PKIPath, server.Namespace, server.Name),
	)
}
