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

package controllers

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cu "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/controller-tools/manager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IamRoleController", func() {
	var mgr manager.IntegrationTest
	var iamService pkgaws.IamService
	var roleService iamrole.Interface
	BeforeEach(func() {
		iamService = newIamService()
		roleService = iamrole.New(iamService, "controller-test")

		c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(c).ToNot(BeNil())

		mgr = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		Expect((&IamRoleReconciler{
			Client:      mgr.GetClient(),
			Scheme:      mgr.GetScheme(),
			notify:      &notifier{},
			RoleService: roleService,
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
				return cu.ContainsFinalizer(obj, Finalizer)
			}).Should(Succeed())
		})
		When("it's being deleted", func() {
			JustBeforeEach(func() {
				mgr.Expect().Delete(instance.DeepCopy()).Should(Succeed())
			})
			When("it has a finalizer", func() {
				BeforeEach(func() {
					mgr.Eventually().GetWhen(key, instance.DeepCopy(), func(obj client.Object) bool {
						return cu.ContainsFinalizer(obj, Finalizer)
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
							return !cu.ContainsFinalizer(obj, Finalizer)
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
							return !cu.ContainsFinalizer(obj, Finalizer)
						}).Should(Succeed())
					})
				})
			})
			When("it doesn't have a finalizer", func() {
				JustBeforeEach(func() {
					mgr.Eventually().GetWhen(key, instance.DeepCopy(), func(obj client.Object) bool {
						return !cu.ContainsFinalizer(obj, Finalizer)
					}).Should(Succeed())
				})
				It("Should not add a finalizer", func() {
					Consistently(func() bool {
						obj := &v1alpha1.IamRole{}
						mgr.Expect().Get(key, obj).Should(Succeed())
						return !cu.ContainsFinalizer(obj, Finalizer)
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
				Expect(instance.Finalizers).Should(And(ContainElement(Finalizer), ContainElement("keep")))
				Expect(instance.ManagedFields[0].Manager).To(Equal("aws-iam-controller"))
				Expect(instance.ManagedFields[1].Manager).ToNot(Equal("aws-iam-controller"))

				managed := map[string]interface{}{
					"f:metadata": map[string]interface{}{
						"f:finalizers": map[string]interface{}{
							fmt.Sprintf(`v:"%s"`, Finalizer): map[string]interface{}{},
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
