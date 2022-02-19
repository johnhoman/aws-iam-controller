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
			Name: "iam-policy",
			Document: `{"Version": "2012-10-17", "Statement": [{"Sid": "S3FullAccess"}]}`,
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
				Name: "iam-policy",
				Document: `{"Version": "2012-10-17", "Statement": [{"Sid": "S3FullAccess"}]}`,
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
	})
})
