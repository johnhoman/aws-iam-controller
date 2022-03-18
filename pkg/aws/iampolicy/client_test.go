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

package iampolicy_test

import (
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/fake"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var service = fake.NewIamService()
	var client iampolicy.Interface
	BeforeEach(func() {
		service.Reset()
		client = iampolicy.New(service, "controller-test")
	})
	It("should create an iam policy", func() {
		out, err := client.Create(ctx, &iampolicy.CreateOptions{
			Name:        "iam-policy",
			Document:    `{"Version": "2012-10-17", "Statement": [{"Sid": "S3FullAccess"}]}`,
			Description: "iam test policy",
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(out).ShouldNot(BeNil())
	})
	When("the policy exists", func() {
		var p *iampolicy.IamPolicy
		BeforeEach(func() {
			var err error
			p, err = client.Create(ctx, &iampolicy.CreateOptions{
				Name:        "iam-policy",
				Document:    `{"Version": "2012-10-17", "Statement": [{"Sid": "S3FullAccess"}]}`,
				Description: "iam test policy",
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(p).ShouldNot(BeNil())
		})
		AfterEach(func() {
			err := client.Delete(ctx, &iampolicy.DeleteOptions{Arn: p.Arn})
			if err != nil {
				Expect(pkgaws.IsNotFound(err)).Should(BeTrue())
			}
		})
		It("should delete the policy", func() {
			err := client.Delete(ctx, &iampolicy.DeleteOptions{Arn: p.Arn})
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("should get the policy", func() {
			out, err := client.Get(ctx, &iampolicy.GetOptions{Arn: p.Arn})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(out.Document).Should(Equal(p.Document))
			Expect(out.CreateDate).Should(Equal(p.CreateDate))
			Expect(out.Id).Should(Equal(p.Id))
			Expect(out.Description).Should(Equal(p.Description))
		})
		It("should get the policy by name", func() {
			out, err := client.Get(ctx, &iampolicy.GetOptions{Name: "iam-policy"})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(out.Document).Should(Equal(p.Document))
			Expect(out.CreateDate).Should(Equal(p.CreateDate))
			Expect(out.Id).Should(Equal(p.Id))
			Expect(out.Description).Should(Equal(p.Description))
		})
		It("should update the policy document", func() {
			doc := `{"Version": "2012-10-17", "Statement": [{"Sid": "S3NoAccess"}]}`
			updated, err := client.Update(ctx, &iampolicy.UpdateOptions{
				Arn:      p.Arn,
				Document: doc,
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updated).ShouldNot(BeNil())
			out, err := client.Get(ctx, &iampolicy.GetOptions{Arn: p.Arn})
			Expect(out.Document).Should(Equal(doc))
			Expect(out.VersionId).ShouldNot(Equal(p.VersionId))
		})
		It("should not update the policy document if unchanged", func() {
			doc := `{"Version": "2012-10-17", "Statement": [{"Sid": "S3FullAccess"}]}`
			updated, err := client.Update(ctx, &iampolicy.UpdateOptions{
				Arn:      p.Arn,
				Document: doc,
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updated).ShouldNot(BeNil())
			out, err := client.Get(ctx, &iampolicy.GetOptions{Arn: p.Arn})
			Expect(out.Document).Should(Equal(doc))
			Expect(out.VersionId).Should(Equal(p.VersionId))
		})
	})
})
