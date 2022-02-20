package iampolicy

import (
	"context"
    "fmt"
    "github.com/aws/aws-sdk-go-v2/aws"
    "k8s.io/apimachinery/pkg/util/json"
    "net/url"
    "reflect"

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
    return c.Get(ctx, &GetOptions{Arn: aws.ToString(out.Policy.Arn)})
}

func (c* Client) Update(ctx context.Context, options *UpdateOptions) (*IamPolicy, error) {
    policy, err := c.Get(ctx, &GetOptions{Arn: options.Arn})
    if err != nil {
        return nil, err
    }
    old := map[string]interface{}{}
    updated := map[string]interface{}{}

    if err := json.Unmarshal([]byte(policy.Document), &old); err != nil {
        return nil, err
    }

    if err := json.Unmarshal([]byte(options.Document), &updated); err != nil {
        return nil, err
    }
    if reflect.DeepEqual(old, updated) {
        // if they're the same then do nothing
        return policy, nil
    }

    _, err = c.service.CreatePolicyVersion(ctx, &iam.CreatePolicyVersionInput{
        PolicyArn: aws.String(policy.Arn),
        PolicyDocument: aws.String(options.Document),
        SetAsDefault: true,
    })
    if err != nil {
        return nil, err
    }

    // If this fails it would require manual intervention
    // this version id isn't tracked once a new version is created so
    // the caller will have to
    _, err = c.service.DeletePolicyVersion(ctx, &iam.DeletePolicyVersionInput{
        PolicyArn: aws.String(policy.Arn),
        VersionId: aws.String(policy.VersionId),
    })
    if err != nil {
        return nil, err
    }

    return c.Get(ctx, &GetOptions{Arn: policy.Arn})
}

func (c* Client) Get(ctx context.Context, options *GetOptions) (*IamPolicy, error) {
    out, err := c.service.GetPolicy(ctx, &iam.GetPolicyInput{
        PolicyArn: aws.String(options.Arn),
    })
    if err != nil {
        return nil, err
    }

    versionOut, err := c.service.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
        PolicyArn: out.Policy.Arn,
        VersionId: out.Policy.DefaultVersionId,
    })
    if err != nil {
        return nil, err
    }

    document, err := url.QueryUnescape(aws.ToString(versionOut.PolicyVersion.Document))
    if err != nil {
        return nil, err
    }

    iamPolicy := &IamPolicy{}
    iamPolicy.CreateDate = aws.ToTime(out.Policy.CreateDate)
    iamPolicy.Arn = aws.ToString(out.Policy.Arn)
    iamPolicy.Description = aws.ToString(out.Policy.Description)
    iamPolicy.Name = aws.ToString(out.Policy.PolicyName)
    iamPolicy.Id = aws.ToString(out.Policy.PolicyId)
    iamPolicy.VersionId = aws.ToString(out.Policy.DefaultVersionId)
    iamPolicy.Document = document

    return iamPolicy, nil
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