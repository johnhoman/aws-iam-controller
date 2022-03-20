package iamrole_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/fake"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/google/uuid"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var service pkgaws.IamService
	var client iamrole.Interface
	var namespace string
	var ctx context.Context
	var policy string
	var role *iamrole.IamRole
	BeforeEach(func() {
		ctx = context.Background()
		namespace = "testspace-" + uuid.New().String()[:8]
		service = fake.NewIamService()
		client = iamrole.New(service, namespace)

		doc := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []interface{}{
				map[string]interface{}{
					"Sid":       "DenyAllAWS",
					"Effect":    "Deny",
					"Principal": map[string]interface{}{"AWS": "*"},
					"Action":    "sts:AssumeRole",
				},
			},
		}
		out, err := json.Marshal(doc)
		Expect(err).ShouldNot(HaveOccurred())
		policy = string(out)
	})
	AfterEach(func() {
		err := client.Delete(ctx, &iamrole.DeleteOptions{Name: role.Name})
		if err != nil {
			expected := &iamtypes.NoSuchEntityException{}
			Expect(errors.As(err, &expected)).To(BeTrue())
		}
	})
	It("Should create a role", func() {
		var err error
		role, err = client.Create(ctx, &iamrole.CreateOptions{
			Name:               "should-create-a-role",
			Description:        "Should create a role",
			MaxDurationSeconds: 3600,
			PolicyDocument:     policy,
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(role.Id).ShouldNot(Equal(""))
		Expect(role.CreateDate.Before(time.Now())).Should(BeTrue())
		Expect(role.Arn).To(HaveSuffix(fmt.Sprintf(
			"%s/should-create-a-role", namespace,
		)))
		upstream, err := service.GetRole(ctx, &iam.GetRoleInput{
			RoleName: aws.String(role.Name),
		})
		p, err := url.QueryUnescape(aws.ToString(upstream.Role.AssumeRolePolicyDocument))
		Expect(err).Should(Succeed())
		var up map[string]interface{}
		var m map[string]interface{}
		Expect(json.Unmarshal([]byte(role.TrustPolicy), &up))
		Expect(json.Unmarshal([]byte(p), &m))
		Expect(m).To(Equal(up))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(aws.ToString(upstream.Role.Arn)).Should(Equal(role.Arn))
		Expect(aws.ToString(upstream.Role.RoleId)).Should(Equal(role.Id))
		Expect(aws.ToString(upstream.Role.Description)).Should(Equal(role.Description))
	})
	It("Should update the role", func() {
		var err error
		role, err = client.Create(ctx, &iamrole.CreateOptions{
			Name:               "should-create-a-role",
			Description:        "Should create a role",
			MaxDurationSeconds: 3600,
			PolicyDocument:     policy,
		})
		updated := &iamrole.IamRole{}
		By("Changing the MaxDurationSeconds", func() {
			updated, err = client.Update(ctx, &iamrole.UpdateOptions{
				Name:               role.Name,
				MaxDurationSeconds: 7200,
			})
			Expect(err).Should(Succeed())
		})
		Expect(updated.Id).Should(Equal(role.Id))
		Expect(updated.Description).Should(Equal(role.Description))
		Expect(updated.TrustPolicy).Should(Equal(role.TrustPolicy))

		upstream, err := service.GetRole(ctx, &iam.GetRoleInput{
			RoleName: aws.String(role.Name),
		})
		p, err := url.QueryUnescape(aws.ToString(upstream.Role.AssumeRolePolicyDocument))
		Expect(err).Should(Succeed())
		var up map[string]interface{}
		var m map[string]interface{}
		Expect(json.Unmarshal([]byte(role.TrustPolicy), &up))
		Expect(json.Unmarshal([]byte(p), &m))
		Expect(m).To(Equal(up))

		Expect(err).ShouldNot(HaveOccurred())
		Expect(aws.ToString(upstream.Role.Arn)).Should(Equal(role.Arn))
		Expect(aws.ToString(upstream.Role.RoleId)).Should(Equal(role.Id))
		Expect(aws.ToString(upstream.Role.Description)).Should(Equal(role.Description))
		Expect(aws.ToInt32(upstream.Role.MaxSessionDuration)).Should(Equal(int32(7200)))
	})
	It("can attach a policy to the role", func() {
		var policyClient = iampolicy.New(service, namespace)
		var err error
		role, err = client.Create(ctx, &iamrole.CreateOptions{
			Name:               fmt.Sprintf("iam-role-%s", uuid.New().String()),
			Description:        "aws iam controller tests",
			MaxDurationSeconds: 3600,
			PolicyDocument:     policy,
		})
		Expect(err).ShouldNot(HaveOccurred())

		p, err := policyClient.Create(ctx, &iampolicy.CreateOptions{
			Name:        "iam-policy",
			Document:    `{"Version": "2012-10-17", "Statement": [{"Sid": "S3FullAccess"}]}`,
			Description: "iam test policy",
		})

		Expect(client.AttachPolicy(ctx, &iamrole.AttachOptions{
			Name:      role.Name,
			PolicyArn: p.Arn,
		})).Should(Succeed())
		Expect(client.DetachPolicy(ctx, &iamrole.DetachOptions{
			Name:      role.Name,
			PolicyArn: p.Arn,
		})).Should(Succeed())
	})
})
