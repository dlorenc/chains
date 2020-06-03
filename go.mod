module github.com/dlorenc/chains

go 1.14

require (
	cloud.google.com/go/storage v1.0.0
	github.com/Azure/go-autorest v14.1.1+incompatible // indirect
	github.com/google/go-containerregistry v0.0.0-20200115214256-379933c9c22b
	github.com/tektoncd/pipeline v0.12.0
	go.uber.org/zap v1.15.0
	gocloud.dev v0.19.0
	golang.org/x/crypto v0.0.0-20200206161412-a0c6ece9d31a
	google.golang.org/grpc v1.29.1 // indirect
	google.golang.org/protobuf v1.21.0 // indirect
	k8s.io/api v0.18.2 // indirect
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20200509234445-b52862b1b3ea
)

// Knative deps (release-0.13)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	knative.dev/caching => knative.dev/caching v0.0.0-20200116200605-67bca2c83dfa
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200306230727-a56a6ea3fa56
	knative.dev/pkg/vendor/github.com/spf13/pflag => github.com/spf13/pflag v1.0.5
)

// Pin k8s deps to 1.16.5
replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
)
