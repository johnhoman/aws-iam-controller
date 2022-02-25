package controllers

import (
	"fmt"
	"github.com/google/uuid"
	awsv1alpha1 "github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/fake"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"
	"github.com/johnhoman/controller-tools/manager"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IamPolicyController", func() {
	var it manager.IntegrationTest
	var service iampolicy.Interface
	BeforeEach(func() {
		service = iampolicy.New(fake.NewIamService(), "controller.test")
		it = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		err := (&IamPolicyReconciler{
			Client:        it.GetClient(),
			Scheme:        it.GetScheme(),
			EventRecorder: it.GetEventRecorderFor("controller.test"),
			AWS:           service,
		}).SetupWithManager(it)
		Expect(err).ShouldNot(HaveOccurred())

		it.StartManager()
	})
	AfterEach(func() { it.StopManager() })
	When("the iam policy exists", func() {
		var instance *awsv1alpha1.IamPolicy
		var key types.NamespacedName
		BeforeEach(func() {
			key = types.NamespacedName{Name: fmt.Sprintf("controller-test-%s", uuid.New().String()[:8])}
			instance = &awsv1alpha1.IamPolicy{}
			instance.SetName(key.Name)
			instance.SetFinalizers([]string{"keep-alive"})
			instance.Spec.Document = awsv1alpha1.IamPolicyDocument{
				Statements: []awsv1alpha1.Statement{{
					Effect:    "Allow",
					Actions:   []string{"s3:ListBucket", "s3:CreateBucket", "s3:DeleteBucket"},
					Resources: []string{"*"},
				}},
			}
			it.Eventually().Create(instance).Should(Succeed())
		})
		It("should have a finalizer", func() {
			policy := &awsv1alpha1.IamPolicy{}
			it.Eventually().GetWhen(key, policy, func(o client.Object) bool {
				return controllerutil.ContainsFinalizer(policy, IamPolicyFinalizer)
			}).Should(Succeed())
		})
	})
})
