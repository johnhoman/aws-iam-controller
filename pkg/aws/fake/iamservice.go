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
	"github.com/aws/smithy-go"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	transporthttp "github.com/aws/smithy-go/transport/http"

	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
)

const (
	AWSAccountId = "012345678912"
)

type IamService struct {
	// RoleName -> Role
	Roles map[string]iamtypes.Role
	// Policies []iamtypes.Policy
}

func (i *IamService) UpdateAssumeRolePolicy(ctx context.Context, params *iam.UpdateAssumeRolePolicyInput, optFns ...func(options *iam.Options)) (*iam.UpdateAssumeRolePolicyOutput, error) {
	in := &iam.GetRoleInput{RoleName: params.RoleName}
	_, err := i.GetRole(ctx, in)
	if err != nil {
		return &iam.UpdateAssumeRolePolicyOutput{}, err
	}
	role := i.Roles[aws.ToString(params.RoleName)]
	role.AssumeRolePolicyDocument = aws.String(url.QueryEscape(*params.PolicyDocument))
	i.Roles[aws.ToString(params.RoleName)] = role
	return &iam.UpdateAssumeRolePolicyOutput{}, nil
}

func (i *IamService) UpdateRole(ctx context.Context, params *iam.UpdateRoleInput, optFns ...func(*iam.Options)) (*iam.UpdateRoleOutput, error) {
	in := &iam.GetRoleInput{RoleName: params.RoleName}
	out, err := i.GetRole(ctx, in)
	if err != nil {
		return &iam.UpdateRoleOutput{}, err
	}
	roleName := aws.ToString(out.Role.RoleName)
	role := i.Roles[roleName]
	if params.Description != nil {
		role.Description = params.Description
	}
	if params.MaxSessionDuration != nil {
		role.MaxSessionDuration = params.MaxSessionDuration
	}
	i.Roles[roleName] = role
	return &iam.UpdateRoleOutput{}, nil
}

func (i *IamService) GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
	re := regexp.MustCompile(`[\w+=,.@-]+`)
	if !re.MatchString(aws.ToString(params.RoleName)) {
		return nil, wrap(&iamtypes.InvalidInputException{
			Message: aws.String("invalid role name"),
		}, &http.Response{StatusCode: http.StatusBadRequest}, "GetRole")
	}

	role, ok := i.Roles[aws.ToString(params.RoleName)]
	if !ok {
		return &iam.GetRoleOutput{}, wrap(
			&iamtypes.NoSuchEntityException{},
			&http.Response{StatusCode: http.StatusNotFound},
			"GetRole",
		)
	}
	return &iam.GetRoleOutput{Role: &role}, nil
}

func (i *IamService) CreateRole(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
	in := &iam.GetRoleInput{RoleName: params.RoleName}
	if _, err := i.GetRole(ctx, in); err == nil {
		return &iam.CreateRoleOutput{}, wrap(
			&iamtypes.EntityAlreadyExistsException{},
			&http.Response{StatusCode: http.StatusConflict},
			"CreateRole",
		)
	}
	if params.AssumeRolePolicyDocument == nil {
		return &iam.CreateRoleOutput{}, &smithy.OperationError{Err: &smithy.InvalidParamsError{}}
	}

	arn := fmt.Sprintf("arn:aws:iam::%s:role", AWSAccountId)
	if params.Path != nil {
		path := aws.ToString(params.Path)
		if path != "/" {
			if !strings.HasPrefix(path, "/") || !strings.HasSuffix(path, "/") {
				return &iam.CreateRoleOutput{}, wrap(
					&iamtypes.InvalidInputException{},
					&http.Response{StatusCode: http.StatusBadRequest},
					"CreateRole",
				)
			}
			arn = arn + "/" + strings.Trim(path, "/")
		}
	}
	arn = arn + "/" + aws.ToString(params.RoleName)

	iamRole := iamtypes.Role{}
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

	i.Roles[aws.ToString(iamRole.RoleName)] = iamRole
	return &iam.CreateRoleOutput{Role: &iamRole}, nil
}

func (i *IamService) DeleteRole(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
	in := &iam.GetRoleInput{RoleName: params.RoleName}
	out, err := i.GetRole(ctx, in)
	if err != nil {
		return &iam.DeleteRoleOutput{}, err
	}
	delete(i.Roles, aws.ToString(out.Role.RoleName))
	// TODO: delete conflicts
	return &iam.DeleteRoleOutput{}, nil
}

var _ pkgaws.IamService = &IamService{}

func NewIamService() *IamService {
	return &IamService{Roles: make(map[string]iamtypes.Role)}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randStringSuffix(p string) string {
	runes := []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 17)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return p + string(b)
}

func wrap(err error, res *http.Response, opName string) error {
	return &smithy.OperationError{
		OperationName: opName,
		Err: &awshttp.ResponseError{
			ResponseError: &transporthttp.ResponseError{
				Response: &transporthttp.Response{Response: res},
				Err:      err,
			},
		},
	}
}
