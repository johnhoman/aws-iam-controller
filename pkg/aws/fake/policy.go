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
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func (i *IamService) ListPolicies(_ context.Context, params *iam.ListPoliciesInput, _ ...func(*iam.Options)) (*iam.ListPoliciesOutput, error) {
	if params == nil {
		params = &iam.ListPoliciesInput{}
	}

	policies := make([]iamtypes.Policy, 0, 10)
	i.ManagedPolicies.Range(func(k interface{}, v interface{}) bool {
		mp := v.(managedPolicy)
		policies = append(policies, mp.policy)
		return true
	})

	return &iam.ListPoliciesOutput{Policies: policies}, nil
}

func (i *IamService) CreatePolicy(_ context.Context, p *iam.CreatePolicyInput, _ ...func(*iam.Options)) (*iam.CreatePolicyOutput, error) {

	if _, ok := i.ManagedPolicies.Load(aws.ToString(p.PolicyName)); ok {
		return nil, &iamtypes.EntityAlreadyExistsException{}
	}

	out := &iam.CreatePolicyOutput{}
	path := aws.String("/")
	if p.Path != nil {
		path = p.Path
	}

	if !strings.HasSuffix(*path, "/") || !strings.HasPrefix(*path, "/") {
		return nil, &iamtypes.InvalidInputException{}
	}

	policy := iamtypes.Policy{
		Arn:              aws.String(fmt.Sprintf("arn:aws:iam::%s:policy%s%s", i.AccountID, *path, *p.PolicyName)),
		AttachmentCount:  aws.Int32(0),
		CreateDate:       aws.Time(time.Now().UTC()),
		DefaultVersionId: aws.String("v1"),
		Description:      p.Description,
		IsAttachable:     true,
		Path:             p.Path,
		PolicyId:         aws.String(randStringSuffix("ANPA")),
		PolicyName:       p.PolicyName,
	}
	out.Policy = &policy

	document := aws.String(url.QueryEscape(aws.ToString(p.PolicyDocument)))
	version := iamtypes.PolicyVersion{
		Document:         document,
		VersionId:        aws.String("v1"),
		IsDefaultVersion: true,
		CreateDate:       aws.Time(time.Now().UTC()),
	}

	mp := managedPolicy{
		versions: &policyVersions{Map: sync.Map{}},
		policy:   policy,
	}
	mp.versions.Store("v1", version)
	i.Cache.ManagedPolicies.Store(aws.ToString(policy.PolicyName), mp)
	i.Cache.policyArnMapping.Store(
		aws.ToString(policy.Arn),
		aws.ToString(policy.PolicyName),
	)
	return out, nil
}

func (i *IamService) CreatePolicyVersion(_ context.Context, in *iam.CreatePolicyVersionInput, _ ...func(*iam.Options)) (*iam.CreatePolicyVersionOutput, error) {
	name, ok := i.Cache.policyArnMapping.Load(aws.ToString(in.PolicyArn))
	if !ok {
		return nil, &iamtypes.NoSuchEntityException{}
	}
	v, _ := i.Cache.ManagedPolicies.Load(name)
	mp := v.(managedPolicy)
	number := aws.ToString(mp.versions.Latest().VersionId)[1:]
	latest, _ := strconv.Atoi(number)

	version := iamtypes.PolicyVersion{
		CreateDate: aws.Time(time.Now().UTC()),
		VersionId:  aws.String(fmt.Sprintf("v%d", latest+1)),
		Document:   in.PolicyDocument,
	}
	if in.SetAsDefault {
		version.IsDefaultVersion = true
	}
	if mp.versions.Len() < 5 {
		if version.IsDefaultVersion {
			mp.versions.Range(func(key, value interface{}) bool {
				v := value.(iamtypes.PolicyVersion)
				if value.(iamtypes.PolicyVersion).IsDefaultVersion {
					iFace, _ := mp.versions.Load(aws.ToString(v.VersionId))
					ver := iFace.(iamtypes.PolicyVersion)
					ver.IsDefaultVersion = false
					mp.versions.Store(aws.ToString(v.VersionId), ver)
					return false
				}
				return true
			})
		}
		mp.versions.Store(aws.ToString(version.VersionId), version)

	} else if mp.versions.Len() == 5 {
		return nil, &iamtypes.LimitExceededException{}
	}
	out := &iam.CreatePolicyVersionOutput{}
	out.PolicyVersion = &version
	if version.IsDefaultVersion {
		// Update the policy to point to the default version id
		mp.policy.DefaultVersionId = version.VersionId
		i.Cache.ManagedPolicies.Store(name, mp)
	}
	return out, nil
}

func (i *IamService) DeletePolicy(_ context.Context, p *iam.DeletePolicyInput, _ ...func(*iam.Options)) (*iam.DeletePolicyOutput, error) {
	name, ok := i.Cache.policyArnMapping.LoadAndDelete(aws.ToString(p.PolicyArn))
	if !ok {
		return nil, &iamtypes.NoSuchEntityException{}
	}

	i.Cache.ManagedPolicies.Delete(name)
	return &iam.DeletePolicyOutput{}, nil
}

func (i *IamService) DeletePolicyVersion(_ context.Context, in *iam.DeletePolicyVersionInput, _ ...func(*iam.Options)) (*iam.DeletePolicyVersionOutput, error) {
	name, ok := i.policyArnMapping.Load(aws.ToString(in.PolicyArn))
	if !ok {
		return nil, &iamtypes.NoSuchEntityException{}
	}
	// Cannot delete the policies default version
	pol, _ := i.ManagedPolicies.Load(name)
	mp := pol.(managedPolicy)
	v, ok := mp.versions.Load(aws.ToString(in.VersionId))
	if !ok {
		return nil, &iamtypes.NoSuchEntityException{}
	}
	if v.(iamtypes.PolicyVersion).IsDefaultVersion {
		return nil, &iamtypes.DeleteConflictException{}
	}
	mp.versions.Delete(aws.ToString(in.VersionId))
	return &iam.DeletePolicyVersionOutput{}, nil
}

func (i *IamService) GetPolicy(_ context.Context, in *iam.GetPolicyInput, _ ...func(*iam.Options)) (*iam.GetPolicyOutput, error) {
	name, ok := i.policyArnMapping.Load(aws.ToString(in.PolicyArn))
	if !ok {
		return nil, &iamtypes.NoSuchEntityException{}
	}
	v, _ := i.ManagedPolicies.Load(name)
	mp := v.(managedPolicy)
	policy := mp.policy
	return &iam.GetPolicyOutput{Policy: &policy}, nil
}

func (i *IamService) GetPolicyVersion(_ context.Context, in *iam.GetPolicyVersionInput, _ ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error) {
	name, ok := i.policyArnMapping.Load(aws.ToString(in.PolicyArn))
	if !ok {
		return nil, &iamtypes.NoSuchEntityException{}
	}
	v, _ := i.ManagedPolicies.Load(name)
	mp := v.(managedPolicy)
	version := &iamtypes.PolicyVersion{}
	mp.versions.Range(func(_ interface{}, v interface{}) bool {
		if aws.ToString(v.(iamtypes.PolicyVersion).VersionId) == aws.ToString(in.VersionId) {
			*version = v.(iamtypes.PolicyVersion)
			return false
		}
		return true
	})
	if *version == (iamtypes.PolicyVersion{}) {
		return nil, &iamtypes.NoSuchEntityException{}
	}
	return &iam.GetPolicyVersionOutput{PolicyVersion: version}, nil
}

var _ pkgaws.IamPolicyService = &IamService{}
