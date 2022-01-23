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
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/controller-tools/manager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IamRoleController", func() {
	var mgr manager.IntegrationTest
	var iamService pkgaws.IamService
	BeforeEach(func() {
		iamService = newIamService()

		mgr = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		Expect((&IamRoleReconciler{
			Client:      mgr.GetClient(),
			Scheme:      mgr.GetScheme(),
			notify:      &notifier{},
			roleService: iamrole.New(iamService, "controller-test"),
		}).SetupWithManager(mgr)).Should(Succeed())
		mgr.StartManager()
	})
	AfterEach(func() { mgr.StopManager() })
	It("Adds a finalizer", func() {
		instance := &v1alpha1.IamRole{}
		instance.SetName("adds-a-finalizer")
		instance.SetFinalizers([]string{"keep"})
		mgr.Expect().Create(instance).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		mgr.Eventually().GetWhen(types.NamespacedName{Name: "adds-a-finalizer"}, instance, func(obj client.Object) bool {
			if len(obj.GetFinalizers()) > 1 {
				return true
			}
			return false
		}).Should(Succeed())
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
		raw, _ := json.Marshal(managed)
		Expect(instance.ManagedFields[0].FieldsV1.Raw).To(Equal(raw))
	})
	It("Adds a finalizer forces ownership", func() {
		Skip("this path can't be reached in the controller currently")
		instance := &v1alpha1.IamRole{}
		instance.SetName("adds-a-finalizer")
		instance.SetFinalizers([]string{"keep", Finalizer})
		mgr.Eventually().Create(instance).Should(Succeed())
		Expect(instance.Finalizers).Should(And(ContainElement(Finalizer), ContainElement("keep")))
		Expect(instance.ManagedFields).To(HaveLen(1))
		Expect(instance.ManagedFields[0].Manager).ToNot(Equal("aws-iam-controller"))

		instance = &v1alpha1.IamRole{}
		mgr.Eventually().GetWhen(types.NamespacedName{Name: "adds-a-finalizer"}, instance, func(obj client.Object) bool {
			return len(obj.GetManagedFields()) > 1
		}).Should(Succeed())
		Expect(instance.Finalizers).Should(And(ContainElement(Finalizer), ContainElement("keep")))
		Expect(instance.ManagedFields).To(HaveLen(2))
		Expect(instance.ManagedFields[0].Manager).To(Equal("aws-iam-controller"))
		Expect(instance.ManagedFields[1].Manager).ToNot(Equal("aws-iam-controller"))
	})
	It("Should remove a finalizer", func() {
		instance := &v1alpha1.IamRole{}
		instance.SetName("remove-a-finalizer")
		instance.SetFinalizers([]string{"keep"})
		mgr.Eventually().Create(instance).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		mgr.Eventually().GetWhen(types.NamespacedName{Name: "remove-a-finalizer"}, instance, func(obj client.Object) bool {
			if len(obj.GetFinalizers()) > 1 {
				return true
			}
			return false
		}).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		instance.SetName("remove-a-finalizer")
		mgr.Expect().Delete(instance).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		mgr.Eventually().GetWhen(types.NamespacedName{Name: "remove-a-finalizer"}, instance, func(obj client.Object) bool {
			if len(obj.GetFinalizers()) == 1 {
				return true
			}
			return false
		}).Should(Succeed())
		Expect(instance.Finalizers).Should(ContainElement("keep"))
		Expect(instance.Finalizers).ShouldNot(ContainElement(Finalizer))
		Expect(instance.ManagedFields[0].Manager).ToNot(Equal("aws-iam-controller"))
	})
	It("Should delete the resource and remove the finalizer", func() {
		instance := &v1alpha1.IamRole{}
		instance.SetName("remove-a-finalizer")
		instance.SetFinalizers([]string{"keep"})

		_, err := iamService.GetRole(mgr.GetContext(), &iam.GetRoleInput{RoleName: aws.String(roleName(instance))})
		oe := &iamtypes.NoSuchEntityException{}
		Expect(errors.As(err, &oe)).To(BeTrue())

		mgr.Eventually().Create(instance).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		mgr.Eventually().GetWhen(types.NamespacedName{Name: "remove-a-finalizer"}, instance, func(obj client.Object) bool {
			if len(obj.GetFinalizers()) > 1 {
				return true
			}
			return false
		}).Should(Succeed())
		Expect(instance.Finalizers).Should(ContainElement(Finalizer))

		_, err = iamService.GetRole(mgr.GetContext(), &iam.GetRoleInput{
			RoleName: aws.String(instance.GetName()),
		})
		Expect(err).ShouldNot(HaveOccurred())

		instance = &v1alpha1.IamRole{}
		instance.SetName("remove-a-finalizer")
		mgr.Expect().Delete(instance).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		mgr.Eventually().GetWhen(types.NamespacedName{Name: "remove-a-finalizer"}, instance, func(obj client.Object) bool {
			return len(obj.GetFinalizers()) == 1
		}).Should(Succeed())
		Expect(instance.Finalizers).Should(ContainElement("keep"))
		Expect(instance.Finalizers).ShouldNot(ContainElement(Finalizer))
		Expect(instance.ManagedFields[0].Manager).ToNot(Equal("aws-iam-controller"))
		Consistently(func() bool {
			_, err = iamService.GetRole(mgr.GetContext(), &iam.GetRoleInput{RoleName: aws.String(instance.GetName())})
			oe = &iamtypes.NoSuchEntityException{}
			return errors.As(err, &oe)
		}).Should(BeTrue())
	})
	It("Should create the upstream role", func() {
		instance := &v1alpha1.IamRole{}
		instance.SetName("app-role")
		mgr.Eventually().Create(instance).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		mgr.Eventually().GetWhen(types.NamespacedName{Name: "app-role"}, instance, func(obj client.Object) bool {
			return instance.Status.RoleArn != ""
		}).ShouldNot(HaveOccurred())

		_, err := iamService.GetRole(mgr.GetContext(), &iam.GetRoleInput{
			RoleName: aws.String(instance.GetName()),
		})
		Expect(err).ShouldNot(HaveOccurred())
	})
})
