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

package iampolicy

import (
	"context"
	"fmt"
	"net/url"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/hashicorp/golang-lru"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"k8s.io/apimachinery/pkg/util/json"
)

const DefaultCacheSize = 128

type Client struct {
	service   pkgaws.IamPolicyService
	path      string
	nameCache *lru.Cache
}

func (c *Client) Create(ctx context.Context, options *CreateOptions) (*IamPolicy, error) {
	out, err := c.service.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyDocument: aws.String(options.Document),
		PolicyName:     aws.String(options.Name),
		Description:    aws.String(options.Description),
		Path:           aws.String(c.path),
	})
	if err != nil {
		return nil, err
	}
	return c.Get(ctx, &GetOptions{Arn: aws.ToString(out.Policy.Arn)})
}

func (c *Client) Update(ctx context.Context, options *UpdateOptions) (*IamPolicy, error) {
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
		PolicyArn:      aws.String(policy.Arn),
		PolicyDocument: aws.String(options.Document),
		SetAsDefault:   true,
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

func (c *Client) findArn(ctx context.Context, options *GetOptions) (string, error) {
	arn, ok := c.nameCache.Get(options.Name)
	if ok {
		return arn.(string), nil
	}
	out, err := c.service.ListPolicies(ctx, &iam.ListPoliciesInput{
		PathPrefix: aws.String(c.path),
	})
	if err != nil {
		return "", err
	}
	for _, policy := range out.Policies {
		if aws.ToString(policy.PolicyName) == options.Name {
			arn := aws.ToString(policy.Arn)
			c.nameCache.Add(options.Name, arn)
			return arn, nil
		}
	}
	return "", nil
}

func (c *Client) Get(ctx context.Context, options *GetOptions) (*IamPolicy, error) {

	if len(options.Name) > 0 {
		// find the ARN
		arn, err := c.findArn(ctx, options)
		if err != nil {
			return nil, err
		}
		if len(arn) == 0 {
			// Not found
			return nil, &iamtypes.NoSuchEntityException{}
		}
		options = &GetOptions{Arn: arn}
	}

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

func (c *Client) Delete(ctx context.Context, options *DeleteOptions) error {
	_, err := c.service.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: aws.String(options.Arn)})
	if err != nil {
		return err
	}
	return nil
}

var _ Interface = &Client{}

func New(service pkgaws.IamPolicyService, path string) *Client {
	cache, _ := lru.New(DefaultCacheSize)
	return &Client{
		service:   service,
		path:      fmt.Sprintf("/%s/", path),
		nameCache: cache,
	}
}
