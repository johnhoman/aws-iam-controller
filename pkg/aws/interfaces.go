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

package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type IamPolicyService interface {
	CreatePolicy(context.Context, *iam.CreatePolicyInput, ...func(*iam.Options)) (*iam.CreatePolicyOutput, error)
	CreatePolicyVersion(context.Context, *iam.CreatePolicyVersionInput, ...func(*iam.Options)) (*iam.CreatePolicyVersionOutput, error)
	DeletePolicy(context.Context, *iam.DeletePolicyInput, ...func(*iam.Options)) (*iam.DeletePolicyOutput, error)
	DeletePolicyVersion(context.Context, *iam.DeletePolicyVersionInput, ...func(*iam.Options)) (*iam.DeletePolicyVersionOutput, error)
	GetPolicy(context.Context, *iam.GetPolicyInput, ...func(*iam.Options)) (*iam.GetPolicyOutput, error)
	GetPolicyVersion(context.Context, *iam.GetPolicyVersionInput, ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error)
}

type IamRoleService interface {
	CreateRole(context.Context, *iam.CreateRoleInput, ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	GetRole(context.Context, *iam.GetRoleInput, ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	UpdateRole(context.Context, *iam.UpdateRoleInput, ...func(*iam.Options)) (*iam.UpdateRoleOutput, error)
	DeleteRole(context.Context, *iam.DeleteRoleInput, ...func(*iam.Options)) (*iam.DeleteRoleOutput, error)

	UpdateAssumeRolePolicy(ctx context.Context, params *iam.UpdateAssumeRolePolicyInput, optFns ...func(options *iam.Options)) (*iam.UpdateAssumeRolePolicyOutput, error)
}

// IamService interfaces with an upstream AWS account to create iam resources
type IamService interface {
	IamRoleService
	IamPolicyService
}
