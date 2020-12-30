package controllers

import (
	"context"
	"fmt"
	"time"

	api "github.com/borchero/meerkat-operator/pkg/api/v1alpha1"
	"github.com/borchero/meerkat-operator/pkg/crypto"
	"github.com/borchero/meerkat-operator/pkg/ovpn"
	vaultapi "github.com/hashicorp/vault/api"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=meerkat.borchero.com,resources=ovpnclients,verbs=get;list;watch;create;update;patch;delete

// OvpnClientReconciler reconciles OvpnClient objects.
type OvpnClientReconciler struct {
	ctclient.Client
	config Config
	vault  *vaultapi.Client
	scheme *runtime.Scheme
	logger *zap.Logger
}

// MustSetupOvpnClientReconciler initializes a new server reconciler and attaches it to the given
// manager. It panics on failure.
func MustSetupOvpnClientReconciler(
	config Config, vault *vaultapi.Client, mgr ctrl.Manager, logger *zap.Logger,
) {
	reconciler := &OvpnClientReconciler{
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

func (r *OvpnClientReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.OvpnClient{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

//-------------------------------------------------------------------------------------------------

const (
	secretKeyOvpnCertificate = "certificate.ovpn"

	annotationKeySerial = "meerkat.borchero.com/serial"
	annotationKeyDirty  = "meerkat.borchero.com/dirty"
)

// Reconcile reconciles the given request.
func (r *OvpnClientReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	logger := r.logger.With(zap.String("name", req.String()))
	logger.Debug("starting reconciliation")

	// First, we get the client
	client := &api.OvpnClient{}
	err := r.Get(ctx, req.NamespacedName, client)
	if err != nil {
		return ctrl.Result{}, ctclient.IgnoreNotFound(err)
	}

	// If the client currently exists, we need to check for deletion
	if !client.DeletionTimestamp.IsZero() {
		// In that case, we need to revoke the certificate if the finalizer still exists
		if controllerutil.ContainsFinalizer(client, finalizerIdentifier) {
			if err := r.revokeCertificate(ctx, client, logger); err != nil {
				logger.Error("failed to revoke certificate", zap.Error(err))
				return ctrl.Result{}, err
			}
			logger.Debug("successfully revoked certificate")
		}
		controllerutil.RemoveFinalizer(client, finalizerIdentifier)
		if err := r.Update(ctx, client); err != nil {
			logger.Error("failed removing finalizer", zap.Error(err))
			return ctrl.Result{}, err
		}
		// Returning here automatically removes the secret
		return ctrl.Result{}, nil
	}

	// If we create/update the client, we add the finalizer
	if !controllerutil.ContainsFinalizer(client, finalizerIdentifier) {
		controllerutil.AddFinalizer(client, finalizerIdentifier)
		if err := r.Update(ctx, client); err != nil {
			logger.Error("failed adding finalizer", zap.Error(err))
			return ctrl.Result{}, err
		}
	}

	// Then, we can create the client's certificate
	if err := r.updateCertificate(ctx, client); err != nil {
		logger.Error("failed to reconcile certificate", zap.Error(err))
		return ctrl.Result{}, err
	}

	logger.Info("reconciliation succeeded")
	return ctrl.Result{}, nil
}

//-------------------------------------------------------------------------------------------------

func (r *OvpnClientReconciler) revokeCertificate(
	ctx context.Context, client *api.OvpnClient, logger *zap.Logger,
) error {
	// First, we need to get the certificate secret
	secret := &corev1.Secret{ObjectMeta: client.ObjectRefCertificateSecret()}
	err := r.Get(ctx, ctclient.ObjectKeyFromObject(secret), secret)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for certificate secret: %s", err)
	}

	if apierrors.IsNotFound(err) {
		// If the certificate secret does not exist, we can simply return
		return nil
	}

	// If the secret does exist, we parse the expiration date
	expirationString, ok := secret.Annotations[annotationKeyExpiresAt]
	if !ok {
		// We can't know if the secret is expired, we don't do anything
		return nil
	}
	expiration, err := time.Parse(time.RFC3339, expirationString)
	if err != nil {
		// We don't know about expiration again
		logger.Warn("revocation skipped due to missing expiration date")
		return nil
	}
	if expiration.Before(time.Now()) {
		// If the expiration is in the past, we can return
		logger.Warn("revocation skipped due to invalid expiration date")
		return nil
	}

	// If the certificate has not expired yet, we need to revoke it. For that, it is required that
	// we have a serial.
	serial, ok := secret.Annotations[annotationKeySerial]
	if !ok {
		// We will never be able to revoke the certificate, so we just return
		logger.Warn("revocation skipped due to missing serial")
		return nil
	}

	// Then, we fetch the associated server to get the correct PKI
	server := &api.OvpnServer{}
	serverRef := ctclient.ObjectKey{Name: client.Spec.ServerName, Namespace: client.Namespace}
	if err := r.Get(ctx, serverRef, server); err != nil {
		return fmt.Errorf("failed to get server associated with client: %s", err)
	}

	// Then, we get the PKI and revoke the certificate with the serial from above
	pki := r.getPKI(server)
	if err := pki.Revoke(serial); err != nil {
		return fmt.Errorf("failed to revoke certificate: %s", err)
	}

	// After doing so, we need to trigger an update of the CRL. We simply trigger a reconciliation
	// of the server by adding an annotation to the secret.
	crl := &corev1.Secret{ObjectMeta: server.ObjectRefCrlSecret()}
	op, err := ctrl.CreateOrUpdate(ctx, r, crl, func() error {
		secret.Annotations = map[string]string{
			annotationKeyDirty: "true",
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to flag CRL secret as dirty: %s", err)
	}
	logger.Debug("flagged CRL as dirty", zap.String("operation", string(op)))
	return nil
}

//-------------------------------------------------------------------------------------------------

func (r *OvpnClientReconciler) updateCertificate(
	ctx context.Context, client *api.OvpnClient,
) error {
	// First, we get the certificate secret
	secret := &corev1.Secret{ObjectMeta: client.ObjectRefCertificateSecret()}
	err := r.Get(ctx, ctclient.ObjectKeyFromObject(secret), secret)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for certificate secret: %s", err)
	}
	if err == nil {
		// If the certificate already exists, we don't do anything. Specifially, we don't
		// automatically renew certificates.
		return nil
	}

	// If we cannot find it, we create the certificate. For that, we first need to find the server
	// which is responsible for the user.
	server := &api.OvpnServer{}
	serverRef := ctclient.ObjectKey{Name: client.Spec.ServerName, Namespace: client.Namespace}
	if err := r.Get(ctx, serverRef, server); err != nil {
		return fmt.Errorf("failed to get server associated with client: %s", err)
	}

	// Then, we can get the correct PKI.
	pki := r.getPKI(server)

	// Afterwards, we can generate the private key and certificate.
	validity := client.Spec.Certificate.Validity.Duration
	if validity == 0 {
		validity = server.Spec.Security.Clients.DefaultedValidity()
	}
	certificate, err := pki.Generate("client", client.Spec.CommonName, validity)
	if err != nil {
		return fmt.Errorf("failed to generate new certificate: %s", err)
	}

	// With the certificate, we can now load the shared TLSAuth parameter and then write the
	// full OVPN certificate.
	sharedSecret := &corev1.Secret{ObjectMeta: server.ObjectRefSharedSecrets()}
	if err := r.Get(ctx, ctclient.ObjectKeyFromObject(sharedSecret), sharedSecret); err != nil {
		return fmt.Errorf("failed to get shared secret to build OVPN certificate: %s", err)
	}

	// Get the TLS auth parameters
	tlsAuth, ok := sharedSecret.Data[secretKeyTa]
	if !ok {
		return fmt.Errorf("shared secret does not contain TLS auth")
	}

	// Render the file
	values := ovpn.CertificateValues{
		Host:     server.Spec.Network.Host,
		Port:     server.Spec.Service.DefaultedPort(),
		Protocol: string(server.Spec.Network.DefaultedProtocol()),
		Security: ovpn.ConfigSecurity{
			Hmac:   string(server.Spec.Security.DefaultedHmac()),
			Cipher: string(server.Spec.Security.DefaultedCipher()),
		},
		Secrets: ovpn.CertificateSecrets{
			TLSClientKey: certificate.PrivateKey,
			TLSClientCrt: certificate.Certificate,
			TLSCaCrt:     certificate.CACertificate,
			TLSAuth:      string(tlsAuth),
		},
	}
	ovpnCert, err := ovpn.GetCertificate(values)
	if err != nil {
		return fmt.Errorf("failed to render OVPN certificate: %s", err)
	}

	// And finally, we can store the certificate in the previously referenced secret
	secret.Annotations = map[string]string{
		annotationKeyExpiresAt: certificate.Expiration.Format(time.RFC3339),
		annotationKeySerial:    certificate.Serial,
	}
	secret.StringData = map[string]string{
		secretKeyOvpnCertificate: ovpnCert,
	}
	if err := ctrl.SetControllerReference(client, secret, r.scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on certificate secret: %s", err)
	}
	if err := r.Create(ctx, secret); err != nil {
		return fmt.Errorf("failed to create secret containing certificate: %s", err)
	}
	return nil
}

//-------------------------------------------------------------------------------------------------

func (r *OvpnClientReconciler) getPKI(server *api.OvpnServer) *crypto.PKI {
	return crypto.NewPKI(
		r.vault, fmt.Sprintf("%s/%s/%s", r.config.PKIPath, server.Namespace, server.Name),
	)
}
