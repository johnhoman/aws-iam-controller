package iamrole

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"net/url"

	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
)

type Client struct {
	service pkgaws.IamRoleService
	path    string
}

func (c *Client) Create(ctx context.Context, options *CreateOptions) (*IamRole, error) {
	rv := &IamRole{}
	out, err := c.service.CreateRole(ctx, &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(options.PolicyDocument),
		RoleName:                 aws.String(options.Name),
		Description:              aws.String(options.Description),
		MaxSessionDuration:       aws.Int32(options.MaxDurationSeconds),
		Path:                     aws.String(fmt.Sprintf("/%s/", c.path)),
	})
	if err != nil {
		return rv, err
	}
	return c.Get(ctx, &GetOptions{Name: aws.ToString(out.Role.RoleName)})
}

func (c *Client) Update(ctx context.Context, options *UpdateOptions) (*IamRole, error) {
	in := &iam.UpdateRoleInput{RoleName: aws.String(options.Name)}
	if len(options.Description) > 0 {
		in.Description = aws.String(options.Description)
	}
	if len(options.PolicyDocument) > 0 {
		// Update the policy document
		_, err := c.service.UpdateAssumeRolePolicy(ctx, &iam.UpdateAssumeRolePolicyInput{
			RoleName:       aws.String(options.Name),
			PolicyDocument: aws.String(options.PolicyDocument),
		})
		if err != nil {
			return &IamRole{}, err
		}
	}
	if options.MaxDurationSeconds > 0 {
		in.MaxSessionDuration = aws.Int32(options.MaxDurationSeconds)
	}

	_, err := c.service.UpdateRole(ctx, in)
	if err != nil {
		return &IamRole{}, err
	}
	return c.Get(ctx, &GetOptions{Name: options.Name})
}

func (c *Client) Get(ctx context.Context, options *GetOptions) (*IamRole, error) {
	out, err := c.service.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(options.Name),
	})
	if err != nil {
		return &IamRole{}, err
	}

	policy, err := url.QueryUnescape(aws.ToString(out.Role.AssumeRolePolicyDocument))
	if err != nil {
		return &IamRole{}, err
	}
	return &IamRole{
		TrustPolicy: policy,
		Arn:         aws.ToString(out.Role.Arn),
		Id:          aws.ToString(out.Role.RoleId),
		CreateDate:  aws.ToTime(out.Role.CreateDate),
		Name:        aws.ToString(out.Role.RoleName),
		Description: aws.ToString(out.Role.Description),
	}, nil
}

func (c *Client) Delete(ctx context.Context, options *DeleteOptions) error {
	_, err := c.service.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(options.Name),
	})
	return err
}

func (c *Client) AttachPolicy(ctx context.Context, options *AttachOptions) error {
	if _, err := c.service.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(options.Name),
		PolicyArn: aws.String(options.PolicyArn),
	}); err != nil {
		return err
	}
	return nil
}

func (c *Client) DetachPolicy(ctx context.Context, options *DetachOptions) error {
	if _, err := c.service.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
		RoleName:  aws.String(options.Name),
		PolicyArn: aws.String(options.PolicyArn),
	}); err != nil {
		return err
	}
	return nil
}

func (c *Client) ListAttachedPolicies(ctx context.Context, options *ListOptions) error {
	return nil
}

var _ Interface = &Client{}

func New(service pkgaws.IamRoleService, path string) *Client {
	return &Client{service: service, path: path}
}
