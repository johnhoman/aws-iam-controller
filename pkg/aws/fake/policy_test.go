package fake_test

import (
    "errors"

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
    BeforeEach(func() {
        inputCache.Reset()
        service.Reset()
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
})