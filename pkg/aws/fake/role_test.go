package fake_test

import (
    "encoding/json"
    "errors"
    "fmt"
    "net/url"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/iam"
    iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
    "github.com/aws/smithy-go"
    "github.com/johnhoman/aws-iam-controller/pkg/aws/fake"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("IamRoleService", func() {
    var iamService = fake.NewIamService()
    BeforeEach(func() {
        iamService.Reset()
    })

    It("should create a role", func() {
        _, err := iamService.CreateRole(ctx, &iam.CreateRoleInput{
            RoleName:                 aws.String("should-create-a-role"),
            AssumeRolePolicyDocument: aws.String("{}"),
        })
        Expect(err).To(Succeed())
    })
    It("should return a conflict when a role exists", func() {
        _, err := iamService.CreateRole(ctx, &iam.CreateRoleInput{
            RoleName:                 aws.String("should-create-a-conflict"),
            AssumeRolePolicyDocument: aws.String("{}"),
        })
        Expect(err).To(Succeed())
        _, err = iamService.CreateRole(ctx, &iam.CreateRoleInput{
            RoleName:                 aws.String("should-create-a-conflict"),
            AssumeRolePolicyDocument: aws.String("{}"),
        })
        Expect(err).ToNot(Succeed())
        var e *iamtypes.EntityAlreadyExistsException
        Expect(errors.As(err, &e))
    })
    It("should return not found when a role doesn't exist", func() {
        _, err := iamService.GetRole(ctx, &iam.GetRoleInput{
            RoleName: aws.String("role-does-not-exist"),
        })
        Expect(err).ToNot(Succeed())
        er := &iamtypes.NoSuchEntityException{}
        Expect(errors.As(err, &er)).To(BeTrue())
    })
    It("should delete a role", func() {
        _, err := iamService.CreateRole(ctx, &iam.CreateRoleInput{
            RoleName:                 aws.String("should-delete-a-role"),
            AssumeRolePolicyDocument: aws.String("{}"),
        })
        Expect(err).To(Succeed())
        _, err = iamService.DeleteRole(ctx, &iam.DeleteRoleInput{
            RoleName: aws.String("should-delete-a-role"),
        })
        Expect(err).To(Succeed())
        _, ok := iamService.Roles.Load("should-delete-a-role")
        Expect(ok).ToNot(BeTrue())
    })
    It("should get the role", func() {
        out, err := iamService.CreateRole(ctx, &iam.CreateRoleInput{
            RoleName:                 aws.String("should-get-a-role"),
            AssumeRolePolicyDocument: aws.String("{}"),
        })
        Expect(err).To(Succeed())
        Expect(aws.ToString(out.Role.RoleName)).To(Equal("should-get-a-role"))
        Expect(aws.ToString(out.Role.Arn)).To(Equal(fmt.Sprintf("arn:aws:iam::%s:role/should-get-a-role", iamService.AccountID)))
        Expect(aws.ToString(out.Role.AssumeRolePolicyDocument)).To(Equal(url.QueryEscape("{}")))
    })
    It("should update the role", func() {
        _, err := iamService.CreateRole(ctx, &iam.CreateRoleInput{
            RoleName:                 aws.String("should-update-the-role"),
            AssumeRolePolicyDocument: aws.String("{}"),
        })
        Expect(err).To(Succeed())
        _, err = iamService.UpdateRole(ctx, &iam.UpdateRoleInput{
            RoleName:    aws.String("should-update-the-role"),
            Description: aws.String("A new description"),
        })
        out, err := iamService.GetRole(ctx, &iam.GetRoleInput{RoleName: aws.String("should-update-the-role")})
        Expect(aws.ToString(out.Role.Description)).To(Equal("A new description"))
    })
    It("should raise an error when a policy document isn't provided to create", func() {
        _, err := iamService.CreateRole(ctx, &iam.CreateRoleInput{
            RoleName: aws.String("should-update-the-role"),
        })
        Expect(err).Should(HaveOccurred())
        oe := &smithy.InvalidParamsError{}
        Expect(errors.As(err, &oe)).To(BeTrue())
    })
    It("should update an assume role policy document", func() {
        _, err := iamService.CreateRole(ctx, &iam.CreateRoleInput{
            RoleName:                 aws.String("should-update-assume-role-policy-document"),
            AssumeRolePolicyDocument: aws.String("{}"),
        })
        Expect(err).ShouldNot(HaveOccurred())

        doc := map[string]interface{}{
            "Version": "2012-10-17",
            "Statement": []interface{}{
                map[string]interface{}{
                    "Effect":    "Allow",
                    "Principal": map[string]interface{}{"AWS": "arn:aws:iam::111122223333:root"},
                    "Action":    "sts:AssumeRole",
                },
            },
        }

        raw, err := json.Marshal(doc)
        Expect(err).ShouldNot(HaveOccurred())
        document := string(raw)

        _, err = iamService.UpdateAssumeRolePolicy(ctx, &iam.UpdateAssumeRolePolicyInput{
            RoleName:       aws.String("should-update-assume-role-policy-document"),
            PolicyDocument: aws.String(document),
        })

        out, err := iamService.GetRole(ctx, &iam.GetRoleInput{
            RoleName: aws.String("should-update-assume-role-policy-document"),
        })
        current, err := url.QueryUnescape(aws.ToString(out.Role.AssumeRolePolicyDocument))
        Expect(err).ToNot(HaveOccurred())

        Expect(current).To(Equal(document))
    })
})
