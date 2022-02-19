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

package fake_test

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Iam Policy", func() {
	var service = fake.NewIamService()
	var inputCache = &policies{}
	var versionCache = &versions{}
	BeforeEach(func() {
		inputCache.Reset()
		service.Reset()
		versionCache.Reset()
	})
	It("should create a role", func() {
		_, err := service.CreatePolicy(ctx, inputCache.Pop("AWSHealthFullAccess"))
		Expect(err).ShouldNot(HaveOccurred())
	})
	It("should return conflict when a role exists", func() {
		in := inputCache.Pop("AWSHealthFullAccess")
		_, err := service.CreatePolicy(ctx, in)
		Expect(err).ShouldNot(HaveOccurred())
		_, err = service.CreatePolicy(ctx, in)
		Expect(err).Should(HaveOccurred())
		expected := &iamtypes.EntityAlreadyExistsException{}
		Expect(errors.As(err, &expected)).Should(BeTrue())
	})
	It("should delete a role", func() {
		policy, err := service.CreatePolicy(ctx, inputCache.Pop("AWSHealthFullAccess"))
		Expect(err).ShouldNot(HaveOccurred())
		out, err := service.DeletePolicy(ctx, &iam.DeletePolicyInput{
			PolicyArn: policy.Policy.Arn,
		})
		Expect(err).Should(BeNil())
		Expect(out).Should(Equal(&iam.DeletePolicyOutput{}))
	})
	It("should return an error when a role doesn't exist", func() {
		out, err := service.DeletePolicy(ctx, &iam.DeletePolicyInput{
			PolicyArn: aws.String("arn:aws:iam::aws:policy/AWSHealthFullAccess"),
		})
		Expect(err).ShouldNot(BeNil())
		er := &iamtypes.NoSuchEntityException{}
		Expect(errors.As(err, &er)).Should(BeTrue())
		Expect(out).Should(BeNil())
	})
	When("the policy exists", func() {
		var p *iamtypes.Policy
		BeforeEach(func() {
			in := inputCache.Pop("AWSHealthFullAccess")
			out, err := service.CreatePolicy(ctx, in)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).ToNot(BeNil())
			p = out.Policy
		})
		It("can get a policy", func() {
			out, err := service.GetPolicy(ctx, &iam.GetPolicyInput{
				PolicyArn: p.Arn,
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(out).ShouldNot(BeNil())
			Expect(*out.Policy.Arn).Should(Equal(*p.Arn))
		})
		It("should create a policy version", func() {
			in := versionCache.Pop("AWSHealthFullAccess")
			out, err := service.CreatePolicyVersion(ctx, &in)
			Expect(err).ShouldNot(HaveOccurred())
			version := out.PolicyVersion
			Expect(aws.ToTime(version.CreateDate).Before(time.Now().UTC()))
			Expect(aws.ToString(version.Document)).Should(Equal(*in.PolicyDocument))
		})
		It("should not delete the last policy version", func() {
			out, err := service.DeletePolicyVersion(ctx, &iam.DeletePolicyVersionInput{
				PolicyArn: p.Arn,
				VersionId: p.DefaultVersionId,
			})
			Expect(err).ShouldNot(Succeed())
			Expect(out).Should(BeNil())
			e := &iamtypes.DeleteConflictException{}
			Expect(errors.As(err, &e)).Should(BeTrue())
		})
		It("should delete a policy version", func() {
			in := versionCache.Pop("AWSHealthFullAccess")
			in.SetAsDefault = true
			out, err := service.CreatePolicyVersion(ctx, &in)
			Expect(err).ShouldNot(HaveOccurred())
			version := out.PolicyVersion
			Expect(aws.ToTime(version.CreateDate).Before(time.Now().UTC()))
			Expect(aws.ToString(version.Document)).Should(Equal(*in.PolicyDocument))

			_, err = service.DeletePolicyVersion(ctx, &iam.DeletePolicyVersionInput{
				PolicyArn: p.Arn,
				VersionId: aws.String("v1"),
			})
			Expect(err).ShouldNot(HaveOccurred())
			_, err = service.DeletePolicyVersion(ctx, &iam.DeletePolicyVersionInput{
				PolicyArn: p.Arn,
				VersionId: aws.String("v2"),
			})
			Expect(err).ShouldNot(Succeed())
			Expect(err).Should(Equal(&iamtypes.DeleteConflictException{}))
		})
		It("should get a policy version", func() {
			out, err := service.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
				PolicyArn: p.Arn,
				VersionId: aws.String("v1"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).ShouldNot(BeNil())
			Expect(*out.PolicyVersion.VersionId).Should(Equal("v1"))
			Expect(out.PolicyVersion.IsDefaultVersion).Should(BeTrue())
		})
		It("should return no found when the policy doesn't exist", func() {
			out, err := service.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
				PolicyArn: aws.String("arn:aws:iam::012345678912:policy/DoesNotExist"),
				VersionId: aws.String("v1"),
			})
			Expect(err).To(HaveOccurred())
			Expect(out).Should(BeNil())
			er := &iamtypes.NoSuchEntityException{}
			Expect(errors.As(err, &er)).To(BeTrue())

		})
		It("should return no found when the policy version doesn't exist", func() {
			out, err := service.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
				PolicyArn: p.Arn,
				VersionId: aws.String("v10"),
			})
			Expect(err).To(HaveOccurred())
			Expect(out).Should(BeNil())
			er := &iamtypes.NoSuchEntityException{}
			Expect(errors.As(err, &er)).To(BeTrue())

		})
	})
})
