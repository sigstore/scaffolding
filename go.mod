module github.com/sigstore/scaffolding

go 1.16

require (
	github.com/go-openapi/runtime v0.21.0
	github.com/go-openapi/strfmt v0.21.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/glog v1.0.0
	github.com/google/certificate-transparency-go v1.1.2
	github.com/google/trillian v1.4.0
	github.com/google/uuid v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/sigstore/fulcio v0.1.2-0.20220110181937-d890471d8047
	github.com/sigstore/rekor v0.4.0
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.23.1
	k8s.io/apimachinery v0.23.1
	k8s.io/client-go v0.23.1
	k8s.io/code-generator v0.22.5
	knative.dev/hack v0.0.0-20220111151514-59b0cf17578e
	knative.dev/pkg v0.0.0-20220112181951-2b23ad111bc2
)
