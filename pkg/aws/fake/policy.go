package fake

import (
    "context"
    "fmt"
    "net/url"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/iam"
    iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func(i *IamService) CreatePolicy(_ context.Context, p *iam.CreatePolicyInput, _ ...func(*iam.Options)) (*iam.CreatePolicyOutput, error) {

    if _, ok := i.ManagedPolicies.Load(aws.ToString(p.PolicyName)); ok {
        return nil, &iamtypes.EntityAlreadyExistsException{}
    }

    out := &iam.CreatePolicyOutput{}
    path := aws.String("/")
    if p.Path != nil { path = p.Path }

    if !strings.HasSuffix(*path, "/") || !strings.HasPrefix(*path, "/") {
        return nil, &iamtypes.InvalidInputException{}
    }

    policy := iamtypes.Policy{
        Arn: aws.String(fmt.Sprintf("arn:aws:iam::%s:policy%s%s", i.AccountID, *path, *p.PolicyName)),
        AttachmentCount: aws.Int32(0),
        CreateDate: aws.Time(time.Now().UTC()),
        DefaultVersionId: aws.String("v1"),
        Description: p.Description,
        IsAttachable: true,
        Path: p.Path,
        PolicyId: aws.String(randStringSuffix("ANPA")),
        PolicyName: p.PolicyName,
    }
    out.Policy = &policy

    document := aws.String(url.QueryEscape(aws.ToString(p.PolicyDocument)))
    version := iamtypes.PolicyVersion{
        Document: document,
        VersionId: aws.String("v1"),
        IsDefaultVersion: true,
        CreateDate: aws.Time(time.Now().UTC()),
    }

    mp := managedPolicy{
        versions: [5]iamtypes.PolicyVersion{version},
        policy: policy,
    }
    i.Cache.ManagedPolicies.Store(aws.ToString(policy.PolicyName), mp)
    return out, nil
}

func(i *IamService) DeletePolicy(_ context.Context, p *iam.DeletePolicyInput, _ ...func(*iam.Options)) (*iam.DeletePolicyOutput, error) {
    parts := strings.Split(aws.ToString(p.PolicyArn), "/")
    name := parts[len(parts)-1]

    _, ok := i.Cache.ManagedPolicies.LoadAndDelete(name)
    if !ok {
        return nil, &iamtypes.NoSuchEntityException{}
    }
    return &iam.DeletePolicyOutput{}, nil
}
