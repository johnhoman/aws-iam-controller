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

package fake

import (
	"context"
	"fmt"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"k8s.io/apimachinery/pkg/util/sets"
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

func (i *IamService) AttachRolePolicy(_ context.Context, params *iam.AttachRolePolicyInput, _ ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error) {
	rv := &iam.AttachRolePolicyOutput{}
	if params == nil {
		params = &iam.AttachRolePolicyInput{}
	}
	key := aws.ToString(params.RoleName)
	v, _ := i.Attachments.LoadOrStore(key, sets.NewString())
	policies := v.(sets.String)
	arn := aws.ToString(params.PolicyArn)
	if !policies.Has(arn) {
		policies.Insert(arn)
	} else {
		// Not sure what to do here
		// I think this is wrong
		return nil, &iamtypes.LimitExceededException{}
	}

	return rv, nil
}

func (i *IamService) DetachRolePolicy(_ context.Context, params *iam.DetachRolePolicyInput, _ ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
	rv := &iam.DetachRolePolicyOutput{}
	if params == nil {
		params = &iam.DetachRolePolicyInput{}
	}

	key := aws.ToString(params.RoleName)
	v, ok := i.Attachments.Load(key)
	if !ok {
		return nil, &iamtypes.NoSuchEntityException{}
	}
	policies := v.(sets.String)
	arn := aws.ToString(params.PolicyArn)
	if !policies.Has(arn) {
		return nil, &iamtypes.NoSuchEntityException{}
	}

	i.Attachments.Store(key, policies.Delete(arn))
	return rv, nil
}

func (i *IamService) ListAttachedRolePolicies(
	_ context.Context,
	params *iam.ListAttachedRolePoliciesInput,
	_ ...func(*iam.Options),
) (*iam.ListAttachedRolePoliciesOutput, error) {
	rv := &iam.ListAttachedRolePoliciesOutput{}

	if params == nil {
		params = &iam.ListAttachedRolePoliciesInput{}
	}

	key := aws.ToString(params.RoleName)
	v, ok := i.Attachments.Load(key)
	if !ok {
		return nil, &iamtypes.NoSuchEntityException{}
	}
	arns := v.(sets.String).List()
	attachments := make([]iamtypes.AttachedPolicy, 0, len(arns))

	for _, arn := range arns {
		v, ok := i.policyArnMapping.Load(arn)
		if !ok {
			// Error
		}
		attachments = append(attachments, iamtypes.AttachedPolicy{
			PolicyArn:  aws.String(arn),
			PolicyName: aws.String(v.(string)),
		})
	}
	rv.AttachedPolicies = attachments
	return rv, nil
}

var _ pkgaws.IamRoleService = &IamService{}
