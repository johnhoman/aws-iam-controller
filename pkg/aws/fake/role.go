package fake

import (
    "context"
    "fmt"
    "net/url"
    "strings"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/iam"
    iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
    "github.com/aws/smithy-go"
)

func (i *IamService) UpdateAssumeRolePolicy(_ context.Context, params *iam.UpdateAssumeRolePolicyInput, _ ...func(options *iam.Options)) (*iam.UpdateAssumeRolePolicyOutput, error) {
    iRole, _ := i.Roles.Load(aws.ToString(params.RoleName))
    role := iRole.(*iamtypes.Role)
    role.AssumeRolePolicyDocument = aws.String(url.QueryEscape(*params.PolicyDocument))
    i.Roles.Store(aws.ToString(params.RoleName), role)
    return &iam.UpdateAssumeRolePolicyOutput{}, nil
}

func (i *IamService) UpdateRole(_ context.Context, params *iam.UpdateRoleInput, _ ...func(*iam.Options)) (*iam.UpdateRoleOutput, error) {
    iRole, _ := i.Roles.Load(aws.ToString(params.RoleName))
    role := iRole.(*iamtypes.Role)
    if params.Description != nil {
        role.Description = params.Description
    }
    if params.MaxSessionDuration != nil {
        role.MaxSessionDuration = params.MaxSessionDuration
    }
    i.Roles.Store(aws.ToString(params.RoleName), role)
    return &iam.UpdateRoleOutput{}, nil
}

func (i *IamService) GetRole(_ context.Context, params *iam.GetRoleInput, _ ...func(*iam.Options)) (*iam.GetRoleOutput, error) {

    iRole, ok := i.Roles.Load(aws.ToString(params.RoleName))
    if !ok {
        return nil, &iamtypes.NoSuchEntityException{}
    }
    return &iam.GetRoleOutput{Role: iRole.(*iamtypes.Role)}, nil
}

func (i *IamService) CreateRole(_ context.Context, params *iam.CreateRoleInput, _ ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
    _, ok := i.Roles.Load(aws.ToString(params.RoleName))
    if ok {
        return nil, &iamtypes.EntityAlreadyExistsException{}
    }

    if params.AssumeRolePolicyDocument == nil {
        return nil, &smithy.InvalidParamsError{}
    }

    arn := fmt.Sprintf("arn:aws:iam::%s:role", i.AccountID)
    path := "/"
    if params.Path != nil {
        // The default path is "/"
        if aws.ToString(params.Path) != path {
            path = aws.ToString(params.Path)
        }
        if !strings.HasPrefix(path, "/") || !strings.HasSuffix(path, "/") {
            return nil, &iamtypes.InvalidInputException{}
        }
    }
    arn = arn + path + aws.ToString(params.RoleName)

    iamRole := &iamtypes.Role{}
    iamRole.RoleId = aws.String(randStringSuffix("AROA"))
    iamRole.RoleName = params.RoleName
    iamRole.Arn = aws.String(arn)
    iamRole.Path = params.Path
    iamRole.Description = params.Description
    if params.AssumeRolePolicyDocument != nil {
        iamRole.AssumeRolePolicyDocument = aws.String(url.QueryEscape(aws.ToString(params.AssumeRolePolicyDocument)))
    }
    iamRole.MaxSessionDuration = params.MaxSessionDuration
    iamRole.PermissionsBoundary = &iamtypes.AttachedPermissionsBoundary{
        PermissionsBoundaryArn: params.PermissionsBoundary,
    }

    i.Roles.Store(aws.ToString(params.RoleName), iamRole)
    return &iam.CreateRoleOutput{Role: iamRole}, nil
}

func (i *IamService) DeleteRole(_ context.Context, params *iam.DeleteRoleInput, _ ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
    _, ok := i.Roles.Load(aws.ToString(params.RoleName))
    if !ok {
        return nil, &iamtypes.NoSuchEntityException{}
    }

    i.Roles.Delete(aws.ToString(params.RoleName))
    return &iam.DeleteRoleOutput{}, nil
}

