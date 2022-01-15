module github.com/johnhoman/aws-iam-controller

go 1.16

require (
	github.com/aws/aws-sdk-go-v2 v1.13.0
	github.com/aws/aws-sdk-go-v2/service/iam v1.16.0
	github.com/aws/smithy-go v1.10.0
	github.com/google/uuid v1.1.2
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/controller-runtime v0.10.0
)
