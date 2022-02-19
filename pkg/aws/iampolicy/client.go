package iampolicy

import (
	"context"
    "fmt"
    "github.com/aws/aws-sdk-go-v2/aws"

    "github.com/aws/aws-sdk-go-v2/service/iam"
    pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
)

type Client struct {
    service pkgaws.IamPolicyService
    path string
}

func (c* Client) Create(ctx context.Context, options *CreateOptions) (*IamPolicy, error) {
    out, err := c.service.CreatePolicy(ctx, &iam.CreatePolicyInput{
        PolicyDocument: aws.String(options.Document),
        PolicyName: aws.String(options.Name),
        Description: aws.String(options.Description),
        Path: aws.String(c.path),
    })
    if err != nil {
        return nil, err
    }
    iamPolicy := &IamPolicy{}
    iamPolicy.CreateDate = aws.ToTime(out.Policy.CreateDate)
    iamPolicy.Arn = aws.ToString(out.Policy.Arn)
    iamPolicy.Description = aws.ToString(out.Policy.Description)
    iamPolicy.Name = aws.ToString(out.Policy.PolicyName)
    iamPolicy.Id = aws.ToString(out.Policy.PolicyId)

    return iamPolicy, nil
}

func (c* Client) Update(ctx context.Context, options *UpdateOptions) (*IamPolicy, error) {
    panic("implement me")
}

func (c* Client) Get(ctx context.Context, options *GetOptions) (*IamPolicy, error) {
    panic("implement me")
}

func (c* Client) Delete(ctx context.Context, options *DeleteOptions) error {
    _, err := c.service.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: aws.String(options.Arn)})
    if err != nil {
        return err
    }
    return nil
}

var _ Interface = &Client{}

func New(service pkgaws.IamPolicyService, path string) *Client {
    return &Client{
        service: service,
        path: fmt.Sprintf("/%s/", path),
    }
}