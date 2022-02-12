package controllers

import (
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/controller-tools/manager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("IamRoleBindingController", func() {
	var it manager.IntegrationTest
	BeforeEach(func() {
		it = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		err := (&IamRoleBindingReconciler{
			Client:        it.GetClient(),
			Scheme:        it.GetScheme(),
			EventRecorder: it.GetEventRecorderFor("controller.test"),
		}).SetupWithManager(it)
		Expect(err).ShouldNot(HaveOccurred())

		it.StartManager()
	})
	AfterEach(func() { it.StopManager() })
	When("the RoleBinding role exists", func() {
		var randomName string
		var binding *v1alpha1.IamRoleBinding
		BeforeEach(func() {
			randomName = "integration-test"
			binding = &v1alpha1.IamRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: randomName,
				},
				Spec: v1alpha1.IamRoleBindingSpec{
					IamRoleRef:        randomName,
					ServiceAccountRef: randomName,
				},
			}
			it.Eventually().Create(binding).Should(Succeed())
		})
		When("the role does exist", func() {
			var iamRole *v1alpha1.IamRole
			var iamRoleArn = "arn:aws:iam::123456789012:role/s3access"
			BeforeEach(func() {
				iamRole = &v1alpha1.IamRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: randomName,
					},
					Spec: v1alpha1.IamRoleSpec{},
				}
				it.Eventually().Create(iamRole).Should(Succeed())
				iamRole.Status.RoleArn = iamRoleArn
				Expect(it.Uncached().Status().Update(it.GetContext(), iamRole)).Should(Succeed())
				it.Eventually().GetWhen(types.NamespacedName{Name: randomName}, &v1alpha1.IamRole{}, func(obj client.Object) bool {
					return obj.(*v1alpha1.IamRole).Status.RoleArn == iamRoleArn
				}).Should(Succeed())
			})
			When("the service account exists", func() {
				var serviceAccount *corev1.ServiceAccount
				BeforeEach(func() {
					serviceAccount = &corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name: randomName,
						},
					}
					it.Eventually().Create(serviceAccount).Should(Succeed())
				})
				It("Annotates the service account", func() {
					instance := &corev1.ServiceAccount{}
					it.Eventually().GetWhen(types.NamespacedName{Name: randomName}, instance, func(obj client.Object) bool {
						return len(obj.GetAnnotations()) > 0
					}).Should(Succeed())
					Expect(instance.GetAnnotations()).To(HaveKeyWithValue(ServiceAccountAnnotation, iamRoleArn))

				})
				It("Updates the status", func() {
					instance := &v1alpha1.IamRoleBinding{}
					it.Eventually().GetWhen(types.NamespacedName{Name: randomName}, instance, func(obj client.Object) bool {
						b := obj.(*v1alpha1.IamRoleBinding)
						return len(b.Status.BoundIamRoleArn) > 0 && len(b.Status.BoundServiceAccountRef) > 0
					}).Should(Succeed())
					Expect(instance.Status.BoundServiceAccountRef).Should(Equal(randomName))
					Expect(instance.Status.BoundIamRoleArn).Should(Equal(iamRoleArn))
				})
			})
		})
		When("the role does not exist", func() {
			When("the service account exists", func() {
				var serviceAccount *corev1.ServiceAccount
				BeforeEach(func() {
					serviceAccount = &corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name: randomName,
						},
					}
					it.Eventually().Create(serviceAccount).Should(Succeed())
				})
				It("does not annotate the service account", func() {
					Consistently(func() map[string]string {
						instance := &corev1.ServiceAccount{}
						err := it.Uncached().Get(it.GetContext(), types.NamespacedName{Name: randomName}, instance)
						if err != nil {
							return map[string]string{ServiceAccountFinalizer: ""}
						}
						return instance.GetAnnotations()
					}).ShouldNot(HaveKey(ServiceAccountFinalizer))
				})
			})
		})
	})
})
