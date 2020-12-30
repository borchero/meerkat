package main

import (
	"context"

	meerkatv1alpha1 "github.com/borchero/meerkat-operator/pkg/api/v1alpha1"
	"github.com/borchero/meerkat-operator/pkg/controllers"
	"github.com/borchero/meerkat-operator/pkg/crypto"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
)

type environment struct {
	Debug                bool
	EnableLeaderElection bool `split_words:"true"`
	Server               controllers.Config
	Vault                crypto.VaultConfig
}

func main() {
	// Setup
	var env environment
	envconfig.MustProcess("", &env)

	var logger *zap.Logger
	var err error
	if env.Debug {
		logger, err = zap.NewDevelopment(zap.AddStacktrace(zap.FatalLevel))
	} else {
		logger, err = zap.NewProduction(zap.AddStacktrace(zap.FatalLevel))
	}
	if err != nil {
		panic(err)
	}

	// Configuration of Kubernetes
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(meerkatv1alpha1.AddToScheme(scheme))

	// Configure Vault
	config := vaultapi.DefaultConfig()
	config.Address = env.Vault.Addr
	if err := config.ConfigureTLS(&vaultapi.TLSConfig{
		CACert:        env.Vault.CaCrt,
		TLSServerName: env.Vault.ServerName,
	}); err != nil {
		panic(err)
	}
	vault, err := vaultapi.NewClient(config)
	if err != nil {
		panic(err)
	}
	go crypto.EnsureTokenUpdated(
		context.Background(), vault, env.Vault.TokenMount, logger.Named("vault"),
	)

	// Setup manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: ":8080",
		LeaderElection:     env.EnableLeaderElection,
		LeaderElectionID:   "meerkat.borchero.com",
	})
	if err != nil {
		panic(err)
	}

	// Setup reconcilers
	controllers.MustSetupOvpnServerReconciler(env.Server, vault, mgr, logger.Named("ovpn-server"))
	controllers.MustSetupOvpnClientReconciler(env.Server, vault, mgr, logger.Named("ovpn-client"))

	// And run
	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Fatal("failed to run manager", zap.Error(err))
	}
}
