module github.com/thetechnick/route42

go 1.13

replace (
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

require (
	github.com/caddyserver/caddy v1.0.4
	github.com/coredns/coredns v1.6.5
	github.com/go-logr/logr v0.1.0
	github.com/google/go-cmp v0.3.1
	github.com/miekg/dns v1.1.22
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/spf13/cobra v0.0.5
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.2.2
)
