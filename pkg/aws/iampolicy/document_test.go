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
	It("can serialize a condition", func() {
		doc := iampolicy.NewDocument()
		doc.SetStatements([]iampolicy.Statement{{
			Effect:   "Allow",
			Action:   []string{"s3:*"},
			Resource: []string{"arn:aws:s3:::BUCKET-NAME"},
			Conditions: &iampolicy.Conditions{
				StringLike: map[string][]string{
					"ec2:InstanceType": {"t1.*", "t2.*", "m3.*"},
				},
			},
		}})
		out, err := doc.Marshal()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(out).Should(ContainSubstring(`"ec2:InstanceType":["t1.*","t2.*","m3.*"]`))
	})
	It("can set the version", func() {
		doc := iampolicy.NewDocument()
		doc.SetVersion("2021-12-10")
		Expect(doc.GetVersion()).Should(Equal("2021-12-10"))
	})
})
