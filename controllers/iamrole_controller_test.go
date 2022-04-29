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
	"fmt"

	"github.com/google/uuid"
	"github.com/johnhoman/controller-tools/manager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cu "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/controllers"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"github.com/johnhoman/aws-iam-controller/pkg/bindmanager"
)

var _ = Describe("IamRoleController", func() {
	var mgr manager.IntegrationTest
	var iamService pkgaws.IamService
	var roleService iamrole.Interface
	BeforeEach(func() {
		iamService = newIamService()
		roleService = iamrole.New(iamService, "controller-test")
		bm := bindmanager.New(
			roleService,
			"arn:aws:iam::111122223333:oidc-provider/oidc.eks.region-code.amazonaws.com/id/EXAMPLED539D4633E53DE1B716D3041E",
		)

		c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(c).ToNot(BeNil())

		mgr = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		policy := defaultPolicy()
		raw, err := json.Marshal(policy)
		Expect(err).To(BeNil())

		Expect((&controllers.IamRoleReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			EventRecorder: mgr.GetEventRecorderFor("controller.test"),
			DefaultPolicy: string(raw),
			RoleService:   roleService,
			Manager:       bm,
		}).SetupWithManager(mgr)).Should(Succeed())
		mgr.StartManager()
	})
	AfterEach(func() { mgr.StopManager() })
	When("the resource exists", func() {
		var name string
		var instance *v1alpha1.IamRole
		var key types.NamespacedName
		BeforeEach(func() {
			name = "the-resource-exists-" + uuid.New().String()[:8]
			key = types.NamespacedName{Name: name}
			instance = &v1alpha1.IamRole{}
			instance.SetName(name)
			instance.SetFinalizers([]string{"keep"})

			mgr.Eventually().Create(instance).Should(Succeed())
			instance = &v1alpha1.IamRole{}
			mgr.Eventually().GetWhen(key, instance, func(obj client.Object) bool {
				return cu.ContainsFinalizer(obj, controllers.Finalizer)
			}).Should(Succeed())
		})
		When("a role binding is created", func() {
			// var serviceAccount *corev1.ServiceAccount
			var iamRoleBinding *v1alpha1.IamRoleBinding
			BeforeEach(func() {
				iamRoleBinding = &v1alpha1.IamRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
					Spec: v1alpha1.IamRoleBindingSpec{
						IamRoleRef:        corev1.LocalObjectReference{Name: name},
						ServiceAccountRef: corev1.LocalObjectReference{Name: name},
					},
				}
				mgr.Eventually().Create(iamRoleBinding).Should(Succeed())
				iamRoleBinding2 := &v1alpha1.IamRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: name + "2",
					},
					Spec: v1alpha1.IamRoleBindingSpec{
						IamRoleRef:        corev1.LocalObjectReference{Name: name},
						ServiceAccountRef: corev1.LocalObjectReference{Name: name + "2"},
					},
				}
				mgr.Eventually().Create(iamRoleBinding2).Should(Succeed())
			})
			It("updates the trust policy", func() {
				Eventually(func() string {
					role, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: name})
					if err != nil {
						return ""
					}
					return role.TrustPolicy
				}).Should(ContainSubstring(
					fmt.Sprintf("system:serviceaccount:%s:%s", iamRoleBinding.GetNamespace(), name),
				))
				iamRole := &v1alpha1.IamRole{}
				mgr.Eventually().GetWhen(types.NamespacedName{Name: instance.GetName()}, iamRole, func(o client.Object) bool {
					return len(o.(*v1alpha1.IamRole).Status.BoundServiceAccounts) > 1
				}).Should(Succeed())
				// This might not always work this way
				Expect(iamRole.Status.BoundServiceAccounts).To(ContainElement(
					corev1.ObjectReference{Name: name, Namespace: iamRoleBinding.GetNamespace()},
				))
				Expect(iamRole.Status.BoundServiceAccounts).To(ContainElement(
					corev1.ObjectReference{Name: name + "2", Namespace: iamRoleBinding.GetNamespace()},
				))
			})
		})
		When("it's being deleted", func() {
			JustBeforeEach(func() {
				mgr.Expect().Delete(instance.DeepCopy()).Should(Succeed())
			})
			When("it has a finalizer", func() {
				BeforeEach(func() {
					mgr.Eventually().GetWhen(key, instance.DeepCopy(), func(obj client.Object) bool {
						return cu.ContainsFinalizer(obj, controllers.Finalizer)
					}).Should(Succeed())
				})
				When("the upstream resource exists", func() {
					BeforeEach(func() {
						Eventually(func() error {
							_, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: instance.GetName()})
							return err
						}).Should(Succeed())
					})
					It("should delete the resource", func() {
						Eventually(func() error {
							_, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: instance.GetName()})
							return err
						}).ShouldNot(Succeed())
					})
					It("Should remove the finalizer", func() {
						mgr.Eventually().GetWhen(key, instance.DeepCopy(), func(obj client.Object) bool {
							return !cu.ContainsFinalizer(obj, controllers.Finalizer)
						}).Should(Succeed())
					})
				})
				When("The upstream resource does not exist", func() {
					BeforeEach(func() {
						Eventually(func() error {
							_, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: instance.GetName()})
							return err
						}).ShouldNot(HaveOccurred())
						Expect(roleService.Delete(mgr.GetContext(), &iamrole.DeleteOptions{
							Name: instance.GetName(),
						})).Should(Succeed())
						Eventually(func() error {
							_, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: instance.GetName()})
							return err
						}).ShouldNot(Succeed())
					})
					It("Should remove the finalizer", func() {
						mgr.Eventually().GetWhen(key, instance.DeepCopy(), func(obj client.Object) bool {
							return !cu.ContainsFinalizer(obj, controllers.Finalizer)
						}).Should(Succeed())
					})
				})
			})
			When("it doesn't have a finalizer", func() {
				JustBeforeEach(func() {
					mgr.Eventually().GetWhen(key, instance.DeepCopy(), func(obj client.Object) bool {
						return !cu.ContainsFinalizer(obj, controllers.Finalizer)
					}).Should(Succeed())
				})
				It("Should not add a finalizer", func() {
					Consistently(func() bool {
						obj := &v1alpha1.IamRole{}
						mgr.Expect().Get(key, obj).Should(Succeed())
						return !cu.ContainsFinalizer(obj, controllers.Finalizer)
					}).Should(BeTrue())
				})
				It("Should not recreate the resource", func() {
					Consistently(func() error {
						_, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: instance.GetName()})
						return err
					}).Should(HaveOccurred())
				})
			})
		})
		When("it's not being deleted", func() {
			It("Adds a finalizer", func() {
				Expect(instance.Finalizers).Should(And(ContainElement(controllers.Finalizer), ContainElement("keep")))
				Expect(instance.ManagedFields[0].Manager).To(Equal("aws-iam-controller"))
				Expect(instance.ManagedFields[1].Manager).ToNot(Equal("aws-iam-controller"))

				managed := map[string]interface{}{
					"f:metadata": map[string]interface{}{
						"f:finalizers": map[string]interface{}{
							fmt.Sprintf(`v:"%s"`, controllers.Finalizer): map[string]interface{}{},
						},
					},
				}
				raw, err := json.Marshal(managed)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(instance.ManagedFields[0].FieldsV1.Raw).To(Equal(raw))
			})
			When("The upstream resource exists", func() {
				BeforeEach(func() {
					Eventually(func() error {
						_, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: instance.GetName()})
						return err
					}).Should(Succeed())
				})
				When("The status is out of sync", func() {
					BeforeEach(func() {
						Eventually(func() error {
							obj := &v1alpha1.IamRole{}
							mgr.Eventually().Get(key, obj).Should(Succeed())
							obj.Status = v1alpha1.IamRoleStatus{}
							return mgr.GetClient().Status().Update(mgr.GetContext(), obj)
						}).Should(Succeed())
					})
					It("Updates the status", func() {
						obj := &v1alpha1.IamRole{}
						mgr.Eventually().GetWhen(key, obj, func(obj client.Object) bool {
							return len(obj.(*v1alpha1.IamRole).Status.RoleArn) > 0
						}).Should(Succeed())
					})
				})
			})
			When("The upstream resource doesn't exist", func() {
				JustBeforeEach(func() {
					Expect(roleService.Delete(mgr.GetContext(), &iamrole.DeleteOptions{
						Name: instance.GetName(),
					})).Should(Succeed())
					Eventually(func() error {
						_, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: instance.GetName()})
						return err
					}).Should(HaveOccurred())

					obj := &v1alpha1.IamRole{}
					mgr.Expect().Get(key, obj).Should(Succeed())

					patch := client.MergeFrom(obj.DeepCopy())
					obj.Spec.Description = "Updating for recreate"
					Expect(mgr.Uncached().Patch(mgr.GetContext(), obj, patch)).Should(Succeed())
					mgr.Eventually().GetWhen(types.NamespacedName{Name: obj.GetName()}, &v1alpha1.IamRole{}, func(o client.Object) bool {
						return o.(*v1alpha1.IamRole).Spec.Description == obj.Spec.Description
					}).Should(Succeed())
				})
				It("Should recreate", func() {
					Eventually(func() error {
						_, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: instance.GetName()})
						return err
					}).ShouldNot(HaveOccurred())
				})
			})
		})
	})
})

var _ = Describe("IamRoleController Policy Refs", func() {
	var mgr manager.IntegrationTest
	var iamService pkgaws.IamService
	var roleService iamrole.Interface
	BeforeEach(func() {
		iamService = newIamService()
		roleService = iamrole.New(iamService, "controller-test")
		bm := bindmanager.New(
			roleService,
			"arn:aws:iam::111122223333:oidc-provider/oidc.eks.region-code.amazonaws.com/id/EXAMPLED539D4633E53DE1B716D3041E",
		)

		c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(c).ToNot(BeNil())

		mgr = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		policy := defaultPolicy()
		raw, err := json.Marshal(policy)
		Expect(err).To(BeNil())

		Expect((&controllers.IamRoleReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			EventRecorder: mgr.GetEventRecorderFor("controller.test"),
			DefaultPolicy: string(raw),
			RoleService:   roleService,
			Manager:       bm,
		}).SetupWithManager(mgr)).Should(Succeed())
		mgr.StartManager()
	})
	AfterEach(func() { mgr.StopManager() })
	When("the resource exists", func() {
		var name string
		var instance *v1alpha1.IamRole
		var key types.NamespacedName
		var policyName string
		BeforeEach(func() {
			policyName = "iam-policy-" + uuid.New().String()[:8]
			name = "the-resource-exists-" + uuid.New().String()[:8]
			key = types.NamespacedName{Name: name}
			instance = &v1alpha1.IamRole{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: v1alpha1.IamRoleSpec{
					PolicyRefs: []corev1.ObjectReference{{Name: policyName}},
				},
			}

			mgr.Eventually().Create(instance).Should(Succeed())
			instance = &v1alpha1.IamRole{}
			mgr.Eventually().GetWhen(key, instance, func(obj client.Object) bool {
				return cu.ContainsFinalizer(obj, controllers.Finalizer)
			}).Should(Succeed())
		})
		When("an iam policy exists", func() {
			var policy *v1alpha1.IamPolicy
			var policyClient iampolicy.Interface
			BeforeEach(func() {
				policyClient = iampolicy.New(iamService, "controller-test")
				p, err := policyClient.Create(mgr.GetContext(), &iampolicy.CreateOptions{
					Name:     policyName,
					Document: "{}",
				})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(p).ShouldNot(BeNil())
				policy = &v1alpha1.IamPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: policyName,
					},
					Spec: v1alpha1.IamPolicySpec{
						Document: v1alpha1.IamPolicyDocument{
							Statements: []v1alpha1.Statement{{
								Effect:    v1alpha1.PolicyStatementEffectAllow,
								Actions:   []string{"s3:ListBucket", "s3:CreateBucket", "s3:DeleteBucket"},
								Resources: []string{"*"},
							}},
						},
					},
				}
				mgr.Eventually().Create(policy).Should(Succeed())
				patch := client.MergeFrom(policy.DeepCopy())
				policy.Status.Arn = p.Arn
				policy.Status.AttachedRoles = []corev1.ObjectReference{{Name: instance.GetName()}}
				Expect(mgr.Uncached().Status().Patch(mgr.GetContext(), policy, patch)).Should(Succeed())
				mgr.Eventually().GetWhen(types.NamespacedName{Name: policyName}, &v1alpha1.IamPolicy{}, func(obj client.Object) bool {
					return len(obj.(*v1alpha1.IamPolicy).Status.AttachedRoles) > 0
				}).Should(Succeed())
			})
			It("should attach the iam policy", func() {
				Eventually(func() iamrole.AttachedPolicies {
					attached, err := roleService.ListAttachedPolicies(mgr.GetContext(), &iamrole.ListOptions{
						Name: instance.GetName(),
					})
					if err != nil {
						return nil
					}
					return attached
				}).Should(HaveLen(1))
			})
			When("a reference is removed", func() {
				BeforeEach(func() {
					Eventually(func() iamrole.AttachedPolicies {
						attached, err := roleService.ListAttachedPolicies(mgr.GetContext(), &iamrole.ListOptions{
							Name: instance.GetName(),
						})
						if err != nil {
							return nil
						}
						return attached
					}).Should(HaveLen(1))
					mgr.Eventually().Get(key, instance).Should(Succeed())
					patch := client.MergeFrom(instance.DeepCopy())
					instance.Spec.PolicyRefs = []corev1.ObjectReference{}
					Expect(mgr.Uncached().Patch(mgr.GetContext(), instance, patch)).Should(Succeed())
					mgr.Eventually().GetWhen(key, &v1alpha1.IamRole{}, func(obj client.Object) bool {
						return len(obj.(*v1alpha1.IamRole).Spec.PolicyRefs) == 0
					}).Should(Succeed())
				})
				It("should detach the policy when the reference is removed", func() {
					Eventually(func() iamrole.AttachedPolicies {
						attached, err := roleService.ListAttachedPolicies(mgr.GetContext(), &iamrole.ListOptions{
							Name: instance.GetName(),
						})
						if err != nil {
							return nil
						}
						return attached
					}).Should(HaveLen(0))
				})

			})
		})
	})
})
