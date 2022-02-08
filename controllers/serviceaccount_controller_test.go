package controllers

import (
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/controller-tools/manager"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceaccountController", func() {
	var iamRole *v1alpha1.IamRole
	var binding *v1alpha1.IamRoleBinding
	var serviceAccount *corev1.ServiceAccount
	var it manager.IntegrationTest
	var serviceAccountName = "controller-test"

	BeforeEach(func() {
		it = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		err := (&ServiceAccountReconciler{
			Client:        it.GetClient(),
			Scheme:        it.GetScheme(),
			EventRecorder: it.GetEventRecorderFor("controller.test"),
		}).SetupWithManager(it)
		Expect(err).ShouldNot(HaveOccurred())

		it.StartManager()
	})
	AfterEach(func() { it.StopManager() })
	When("The iam role exists", func() {
		BeforeEach(func() {
			iamRole = &v1alpha1.IamRole{}
			iamRole.SetName("iam-role-name-with-path")
			iamRole.Spec.Description = "the iam role exists"
			iamRole.Spec.MaxDurationSeconds = 7200

			it.Eventually().Create(iamRole).Should(Succeed())
			iamRole.Status.RoleArn = "arn:aws:iam::0123456789012:role/iam-role-name-with-path"
			Expect(it.Uncached().Status().Update(it.GetContext(), iamRole)).Should(Succeed())
			it.Eventually().GetWhen(types.NamespacedName{Name: iamRole.GetName()}, iamRole, func(obj client.Object) bool {
				return len(obj.(*v1alpha1.IamRole).Status.RoleArn) > 0
			}).Should(Succeed())
		})
		When("The role binding exists", func() {
			BeforeEach(func() {
				binding = &v1alpha1.IamRoleBinding{}
				binding.SetName("iam-role-name-with-path")
				binding.Spec.ServiceAccountRef = serviceAccountName
				binding.Spec.IamRoleRef = iamRole.GetName()

				it.Eventually().Create(binding).Should(Succeed())
			})
			When("The service account is created", func() {
				BeforeEach(func() {
					serviceAccount = &corev1.ServiceAccount{}
					serviceAccount.SetName(serviceAccountName)

					it.Eventually().Create(serviceAccount).Should(Succeed())
				})
				It("annotates the service account", func() {
					instance := &corev1.ServiceAccount{}
					it.Eventually().GetWhen(types.NamespacedName{Name: serviceAccountName}, instance, func(obj client.Object) bool {
						return metav1.HasAnnotation(obj.(*corev1.ServiceAccount).ObjectMeta, RoleArnAnnotation)
					}).Should(Succeed())
					Expect(instance.GetAnnotations()).To(HaveKeyWithValue(RoleArnAnnotation, iamRole.Status.RoleArn))
				})
				When("the service account annotation is removed", func() {
					BeforeEach(func() {
						instance := &corev1.ServiceAccount{}
						it.Eventually().GetWhen(types.NamespacedName{Name: serviceAccountName}, instance, func(obj client.Object) bool {
							return metav1.HasAnnotation(obj.(*corev1.ServiceAccount).ObjectMeta, RoleArnAnnotation)
						}).Should(Succeed())
						Expect(instance.GetAnnotations()).To(HaveKeyWithValue(RoleArnAnnotation, iamRole.Status.RoleArn))
						patch := client.MergeFrom(instance.DeepCopy())
						instance.SetAnnotations(map[string]string{})
						Expect(it.Uncached().Patch(it.GetContext(), instance, patch)).Should(Succeed())
					})
					It("annotates the service account", func() {
						instance := &corev1.ServiceAccount{}
						it.Eventually().GetWhen(types.NamespacedName{Name: serviceAccountName}, instance, func(obj client.Object) bool {
							return metav1.HasAnnotation(obj.(*corev1.ServiceAccount).ObjectMeta, RoleArnAnnotation)
						}).Should(Succeed())
						Expect(instance.GetAnnotations()).To(HaveKeyWithValue(RoleArnAnnotation, iamRole.Status.RoleArn))
					})
				})
				When("the role binding is deleted", func() {
					BeforeEach(func() {
						it.Eventually().GetWhen(types.NamespacedName{Name: serviceAccountName}, &corev1.ServiceAccount{}, func(obj client.Object) bool {
							return metav1.HasAnnotation(obj.(*corev1.ServiceAccount).ObjectMeta, RoleArnAnnotation)
						}).Should(Succeed())
						it.Expect().Delete(binding).Should(Succeed())
					})
					It("removes the annotation", func() {
						it.Eventually().GetWhen(types.NamespacedName{Name: serviceAccountName}, &corev1.ServiceAccount{}, func(obj client.Object) bool {
							return !metav1.HasAnnotation(obj.(*corev1.ServiceAccount).ObjectMeta, RoleArnAnnotation)
						}).Should(Succeed())
					})
				})
			})
		})
		When("the role binding doesn't exist", func() {
			When("the service account is annotated", func() {
				JustBeforeEach(func() {
					serviceAccount = &corev1.ServiceAccount{}
					serviceAccount.SetName(serviceAccountName)
					serviceAccount.SetAnnotations(map[string]string{RoleArnAnnotation: iamRole.Status.RoleArn})

					it.Eventually().Create(serviceAccount).Should(Succeed())
				})
				It("should not remove the annotation", func() {
					instance := &corev1.ServiceAccount{}
					it.Eventually().GetWhen(types.NamespacedName{Name: serviceAccountName}, instance, func(obj client.Object) bool {
						return !metav1.HasAnnotation(obj.(*corev1.ServiceAccount).ObjectMeta, RoleArnAnnotation)
					}).ShouldNot(Succeed())
				})
			})
		})
	})
})
