/*
Copyright 2022 John Homan

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	awsv1alpha1 "github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/fake"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

/*
{
  "Effect": "Allow",
  "Principal": {
    "Federated": "arn:aws:iam::your-account-id:oidc-provider/oidc.eks.your-region-code.amazonaws.com/id/EXAMPLE_OIDC_IDENTIFIER"
  },
  "Action": "sts:AssumeRoleWithWebIdentity",
  "Condition": {
    "StringEquals": {
      "oidc.eks.your-region-code.amazonaws.com/id/EXAMPLE_OIDC_IDENTIFIER:sub": "system:serviceaccount:your-namespace:your-service-account"
    }
  }
}
*/

var cfg *rest.Config
var testEnv *envtest.Environment

func defaultPolicy() map[string]interface{} {
	doc := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []interface{}{
			map[string]interface{}{
				"Sid":       "DenyAllAWS",
				"Effect":    "Deny",
				"Principal": map[string]interface{}{"AWS": "*"},
				"Action":    "sts:AssumeRole",
			},
		},
	}
	return doc
}
func newIamService() aws.IamService { // nolint: ireturn
	if _, ok := os.LookupEnv("AWS_IAM_CONTROLLER_AWS_LIVE"); ok {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		Expect(err).ShouldNot(HaveOccurred())
		c := iam.NewFromConfig(cfg)
		return c
	}
	return fake.NewIamService()
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = awsv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = awsv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
