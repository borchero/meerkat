module github.com/borchero/meerkat-operator

go 1.15

require (
	github.com/Masterminds/sprig/v3 v3.2.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/hashicorp/vault/api v1.0.4
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/markbates/pkger v0.17.1
	go.uber.org/zap v1.15.0
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0
	sigs.k8s.io/controller-runtime v0.7.0
)
