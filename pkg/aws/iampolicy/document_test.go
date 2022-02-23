package iampolicy_test

import (
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Document", func() {
	It("should marshal a policy document", func() {
		doc := iampolicy.NewDocument()
		doc.SetStatements([]iampolicy.Statement{{
			Sid:      "AllowS3Access",
			Effect:   "Allow",
			Action:   "s3:*",
			Resource: "arn:aws:s3:::BUCKET-NAME",
		}})
		out, err := doc.Marshal()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(out).ShouldNot(BeNil())
		Expect(out).Should(Equal(
			`{"Version":"2012-10-17","Statement":[{"Sid":"AllowS3Access","Effect":"Allow","Action":["s3:*"],"Resource":["arn:aws:s3:::BUCKET-NAME"]}]}`,
		))
	})
	It("should compare two documents", func() {
		doc, err := iampolicy.NewDocumentFromString(
			`{"Version":"2012-10-17","Statement":[{"Sid":"AllowS3Access","Effect":"Allow","Action":"s3:*","Resource":"arn:aws:s3:::BUCKET-NAME"}]}`,
		)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(doc).ShouldNot(BeNil())
		doc2, err := iampolicy.NewDocumentFromString(
			`{"Version":"2012-10-17","Statement":[{"Sid":"AllowS3Access","Effect":"Allow","Action":"s3:*","Resource":"arn:aws:s3:::BUCKET-NAME"}]}`,
		)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(doc2).ShouldNot(BeNil())

		Expect(doc2.Equals(doc)).Should(BeTrue())
		Expect(doc.Equals(doc2)).Should(BeTrue())
	})
	It("should consider a resource list of length 1 to be the same as a string", func() {
		doc, err := iampolicy.NewDocumentFromString(
			`{"Version":"2012-10-17","Statement":[{"Sid":"AllowS3Access","Effect":"Allow","Action":"s3:*","Resource":"arn:aws:s3:::BUCKET-NAME"}]}`,
		)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(doc).ShouldNot(BeNil())
		doc2, err := iampolicy.NewDocumentFromString(
			`{"Version":"2012-10-17","Statement":[{"Sid":"AllowS3Access","Effect":"Allow","Action":"s3:*","Resource":["arn:aws:s3:::BUCKET-NAME"]}]}`,
		)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(doc2).ShouldNot(BeNil())

		Expect(doc2.Equals(doc)).Should(BeTrue())
		Expect(doc.Equals(doc2)).Should(BeTrue())
	})
})
