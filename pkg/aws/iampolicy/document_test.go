package iampolicy_test

import (
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Document", func() {
	It("should marshal a policy document", func() {
		doc := iampolicy.NewDocument()
		doc.AddStatement(iampolicy.Statement{
			Sid: "AllowS3Access",
			Effect: "Allow",
			Action: "s3:*",
			Resource: "arn:aws:s3:::BUCKET-NAME",
		})
		out, err := doc.Marshal()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(out).ShouldNot(BeNil())
		Expect(out).Should(Equal(
			`{"Version":"2012-10-17","Statement":[{"Sid":"AllowS3Access","Effect":"Allow","Action":"s3:*","Resource":"arn:aws:s3:::BUCKET-NAME"}]}`,
		))
	})
})
