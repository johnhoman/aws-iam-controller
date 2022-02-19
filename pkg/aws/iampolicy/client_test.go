package iampolicy_test

import (
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
			Name: "",
			Document: "",
			Description: "",
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(out).ShouldNot(BeNil())
	})
})
