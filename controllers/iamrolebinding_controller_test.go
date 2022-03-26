package controllers_test

import (
	"github.com/johnhoman/controller-tools/manager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/controllers"
)

var _ = Describe("IamRoleBindingController", func() {
	var it manager.IntegrationTest
	BeforeEach(func() {
		it = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		err := (&controllers.IamRoleBindingReconciler{
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
					IamRoleRef:        corev1.LocalObjectReference{Name: randomName},
					ServiceAccountRef: corev1.LocalObjectReference{Name: randomName},
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
				It("annotates the service account", func() {
					instance := &corev1.ServiceAccount{}
					it.Eventually().GetWhen(types.NamespacedName{Name: randomName}, instance, func(obj client.Object) bool {
						return len(obj.GetAnnotations()) > 0
					}).Should(Succeed())
					Expect(instance.GetAnnotations()).To(HaveKeyWithValue(controllers.ServiceAccountAnnotation, iamRoleArn))

				})
				It("updates the status", func() {
					instance := &v1alpha1.IamRoleBinding{}
					it.Eventually().GetWhen(types.NamespacedName{Name: randomName}, instance, func(obj client.Object) bool {
						b := obj.(*v1alpha1.IamRoleBinding)
						return len(b.Status.BoundIamRoleArn) > 0 && len(b.Status.BoundServiceAccountRef.Name) > 0
					}).Should(Succeed())
					Expect(instance.Status.BoundServiceAccountRef.Name).Should(Equal(randomName))
					Expect(instance.Status.BoundIamRoleArn).Should(Equal(iamRoleArn))
				})
				When("the role binding is deleted", func() {
					BeforeEach(func() {
						it.Eventually().GetWhen(
							types.NamespacedName{Name: randomName},
							&corev1.ServiceAccount{},
							func(obj client.Object) bool {
								meta := obj.(*corev1.ServiceAccount).ObjectMeta
								return metav1.HasAnnotation(meta, controllers.ServiceAccountAnnotation)
							},
						).Should(Succeed())
					})
					JustBeforeEach(func() {
						it.Expect().Delete(binding).Should(Succeed())
					})
					It("removes the service account annotation", func() {
						instance := &corev1.ServiceAccount{}
						it.Eventually().GetWhen(
							types.NamespacedName{Name: randomName},
							instance,
							func(obj client.Object) bool {
								meta := obj.(*corev1.ServiceAccount).ObjectMeta
								return !metav1.HasAnnotation(meta, controllers.ServiceAccountAnnotation)
							},
						).Should(Succeed())
					})
				})
				When("the service account ref is changed", func() {
					var other *corev1.ServiceAccount
					BeforeEach(func() {
						namespacedName := types.NamespacedName{Name: randomName}
						it.Eventually().GetWhen(
							namespacedName,
							&corev1.ServiceAccount{},
							func(obj client.Object) bool {
								meta := obj.(*corev1.ServiceAccount).ObjectMeta
								return metav1.HasAnnotation(meta, controllers.ServiceAccountAnnotation)
							},
						).Should(Succeed())
						other = &corev1.ServiceAccount{
							ObjectMeta: metav1.ObjectMeta{Name: "other"},
						}
						it.Eventually().Create(other).Should(Succeed())

						instance := &v1alpha1.IamRoleBinding{}
						it.Eventually().Get(namespacedName, instance).Should(Succeed())
						patch := client.MergeFrom(instance.DeepCopy())
						instance.Spec.ServiceAccountRef.Name = "other"
						Expect(it.Uncached().Patch(it.GetContext(), instance, patch)).Should(Succeed())
						it.Eventually().GetWhen(
							namespacedName,
							instance,
							func(o client.Object) bool {
								ref := o.(*v1alpha1.IamRoleBinding).Spec.ServiceAccountRef
								return ref.Name == "other"
							},
						).Should(Succeed())
					})
					It("removes the service account annotation from the old service account", func() {
						instance := &corev1.ServiceAccount{}
						it.Eventually().GetWhen(types.NamespacedName{Name: randomName}, instance, func(obj client.Object) bool {
							meta := obj.(*corev1.ServiceAccount).ObjectMeta
							return !metav1.HasAnnotation(meta, controllers.ServiceAccountAnnotation)
						}).Should(Succeed())
					})
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
							return map[string]string{controllers.ServiceAccountFinalizer: ""}
						}
						return instance.GetAnnotations()
					}).ShouldNot(HaveKey(controllers.ServiceAccountFinalizer))
				})
			})
		})
	})
})
