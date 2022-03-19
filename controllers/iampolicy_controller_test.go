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
	awsv1alpha1 "github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/fake"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"
	"github.com/johnhoman/controller-tools/manager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
					Effect:    awsv1alpha1.PolicyStatementEffectAllow,
					Actions:   []string{"s3:ListBucket", "s3:CreateBucket", "s3:DeleteBucket"},
					Resources: []string{"*"},
				}},
			}
			it.Eventually().Create(instance).Should(Succeed())
		})
		It("should have a finalizer", func() {
			policy := &awsv1alpha1.IamPolicy{}
			it.Eventually().GetWhen(key, policy, func(o client.Object) bool {
				return controllerutil.ContainsFinalizer(o, IamPolicyFinalizer)
			}).Should(Succeed())
		})
		It("should update the status", func() {
			policy := &awsv1alpha1.IamPolicy{}
			it.Eventually().GetWhen(key, policy, func(obj client.Object) bool {
				return len(obj.(*awsv1alpha1.IamPolicy).Status.Arn) > 0
			}).Should(Succeed())
			Expect(policy.Status.Md5Sum).ShouldNot(Equal(""))
		})
		It("should convert the conditionals", func() {
			policy := &awsv1alpha1.IamPolicy{}
			it.Eventually().GetWhen(key, policy, func(obj client.Object) bool {
				return len(obj.(*awsv1alpha1.IamPolicy).Status.Arn) > 0
			}).Should(Succeed())
			sum := policy.Status.Md5Sum
			Expect(sum).ShouldNot(Equal(""))
			patch := client.MergeFrom(policy.DeepCopy())
			policy.Spec.Document.Statements[0].Conditions = &awsv1alpha1.Conditions{
				ArnLike: []awsv1alpha1.Condition{{
					Key:    "aws:SourceArn",
					Values: []string{"arn:aws:iam::0123456789012:role/iam-user"},
				}},
			}
			Expect(it.Uncached().Patch(it.GetContext(), policy, patch)).Should(Succeed())
			it.Eventually().GetWhen(key, policy, func(obj client.Object) bool {
				return obj.(*awsv1alpha1.IamPolicy).Status.Md5Sum != sum
			}).Should(Succeed())
		})
		When("the iam policy is marked for deletion", func() {
			var upstream *iampolicy.IamPolicy
			BeforeEach(func() {
				it.Eventually().GetWhen(key, &awsv1alpha1.IamPolicy{}, func(obj client.Object) bool {
					return controllerutil.ContainsFinalizer(obj, IamPolicyFinalizer)
				}).Should(Succeed())
				// setup for upstream resource deletion test
				policy := &awsv1alpha1.IamPolicy{}
				it.Eventually().GetWhen(key, policy, func(obj client.Object) bool {
					return len(obj.(*awsv1alpha1.IamPolicy).Status.Arn) > 0
				}).Should(Succeed())
				var err error
				upstream, err = service.Get(it.GetContext(), &iampolicy.GetOptions{Arn: policy.Status.Arn})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(upstream).ShouldNot(BeNil())
			})
			JustBeforeEach(func() {
				it.Expect().Delete(instance).Should(Succeed())
			})
			It("should remove the finalizer", func() {
				policy := &awsv1alpha1.IamPolicy{}
				it.Eventually().GetWhen(key, policy, func(o client.Object) bool {
					return !controllerutil.ContainsFinalizer(o, IamPolicyFinalizer)
				}).Should(Succeed())
				Expect(policy.GetFinalizers()).Should(ContainElement("keep-alive"))
			})
			It("deletes the resource", func() {
				Eventually(func() error {
					_, err := service.Get(it.GetContext(), &iampolicy.GetOptions{
						Arn: upstream.Arn,
					})
					return err
				}).Should(HaveOccurred())
			})
		})
	})
})
