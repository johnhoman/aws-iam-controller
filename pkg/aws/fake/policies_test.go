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

package fake_test

import (
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	set "github.com/deckarep/golang-set"
)

type policies struct {
	cache sync.Map
}

func (p *policies) Pop(key string) *iam.CreatePolicyInput {
	out, ok := p.cache.LoadAndDelete(key)
	if !ok {
		panic("no such key " + key)
	}
	in := out.(iam.CreatePolicyInput)
	return &in
}

func (p *policies) Reset() {
	p.cache = sync.Map{}
	p.cache.Store("ClientVPNServiceRolePolicy", iam.CreatePolicyInput{
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ec2:CreateNetworkInterface","ec2:CreateNetworkInterfacePermission","ec2:DescribeSecurityGroups","ec2:DescribeVpcs","ec2:DescribeSubnets","ec2:DescribeInternetGateways","ec2:ModifyNetworkInterfaceAttribute","ec2:DeleteNetworkInterface","ec2:DescribeAccountAttributes","ds:AuthorizeApplication","ds:DescribeDirectories","ds:GetDirectoryLimits","ds:UnauthorizeApplication","logs:DescribeLogStreams","logs:CreateLogStream","logs:PutLogEvents","logs:DescribeLogGroups","acm:GetCertificate","acm:DescribeCertificate","iam:GetSAMLProvider","lambda:GetFunctionConfiguration"],"Resource":"*"}]}`),
		PolicyName:     aws.String("ClientVPNServiceRolePolicy"),
		Description:    aws.String(" Policy to enable AWS Client VPN to manage your Client VPN endpoints."),
		Path:           aws.String("/aws-service-role/"),
	})

	p.cache.Store("AWSHealthFullAccess", iam.CreatePolicyInput{
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["organizations:EnableAWSServiceAccess","organizations:DisableAWSServiceAccess"],"Resource":"*","Condition":{"StringEquals":{"organizations:ServicePrincipal":"health.amazonaws.com"}}},{"Effect":"Allow","Action":["health:*","organizations:ListAccounts","organizations:ListParents","organizations:DescribeAccount","organizations:ListDelegatedAdministrators"],"Resource":"*"},{"Effect":"Allow","Action":"iam:CreateServiceLinkedRole","Resource":"*","Condition":{"StringEquals":{"iam:AWSServiceName":"health.amazonaws.com"}}}]}`),
		PolicyName:     aws.String("AWSHealthFullAccess"),
		Description:    aws.String(" Allows full access to the AWS Health Apis and Notifications and the Personal Health Dashboard"),
	})
}

type versions struct {
	cache sync.Map
}

func (v *versions) Pop(name string) iam.CreatePolicyVersionInput {
	remaining, _ := v.cache.Load(name)
	s := remaining.(set.Set)
	rv := s.Pop()
	return *(rv.(*iam.CreatePolicyVersionInput))
}

func (v *versions) Reset() {
	pVersions := set.NewSet()
	pVersions.Add(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String("arn:aws:iam::012345678912:policy/AWSHealthFullAccess"),
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["health:*"],"Resource":"*"}]}`),
		SetAsDefault:   true,
	})
	pVersions.Add(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String("arn:aws:iam::012345678912:policy/AWSHealthFullAccess"),
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["organizations:EnableAWSServiceAccess","organizations:DisableAWSServiceAccess"],"Resource":"*","Condition":{"StringEquals":{"organizations:ServicePrincipal":"health.amazonaws.com"}}},{"Effect":"Allow","Action":["health:*","organizations:ListAccounts"],"Resource":"*"},{"Effect":"Allow","Action":"iam:CreateServiceLinkedRole","Resource":"*","Condition":{"StringEquals":{"iam:AWSServiceName":"health.amazonaws.com"}}}]}`),
		SetAsDefault:   true,
	})
	pVersions.Add(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String("arn:aws:iam::012345678912:policy/AWSHealthFullAccess"),
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["organizations:EnableAWSServiceAccess","organizations:DisableAWSServiceAccess"],"Resource":"*","Condition":{"StringEquals":{"organizations:ServicePrincipal":"health.amazonaws.com"}}},{"Effect":"Allow","Action":["health:*","organizations:ListAccounts","organizations:ListParents","organizations:DescribeAccount","organizations:ListDelegatedAdministrators"],"Resource":"*"},{"Effect":"Allow","Action":"iam:CreateServiceLinkedRole","Resource":"*","Condition":{"StringEquals":{"iam:AWSServiceName":"health.amazonaws.com"}}}]}`),
		SetAsDefault:   true,
	})
	v.cache.Store("AWSHealthFullAccess", pVersions)

	pVersions = set.NewSet()
	pVersions.Add(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String("arn:aws:iam::012345678912:policy/aws-service-role/ClientVPNServiceRolePolicy"),
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ec2:CreateNetworkInterface","ec2:CreateNetworkInterfacePermission","ec2:DescribeSecurityGroups","ec2:DescribeVpcs","ec2:DescribeSubnets","ec2:DescribeInternetGateways","ec2:ModifyNetworkInterfaceAttribute","ec2:DeleteNetworkInterface","ds:AuthorizeApplication","ds:DescribeDirectories","ds:GetDirectoryLimits","ds:ListAuthorizedApplications","ds:UnauthorizeApplication","logs:DescribeLogStreams","logs:CreateLogStream","logs:PutLogEvents","logs:DescribeLogGroups","acm:GetCertificate"],"Resource":"*"}]}`),
		SetAsDefault:   true,
	})
	pVersions.Add(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String("arn:aws:iam::012345678912:policy/aws-service-role/ClientVPNServiceRolePolicy"),
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ec2:CreateNetworkInterface","ec2:CreateNetworkInterfacePermission","ec2:DescribeSecurityGroups","ec2:DescribeVpcs","ec2:DescribeSubnets","ec2:DescribeInternetGateways","ec2:ModifyNetworkInterfaceAttribute","ec2:DeleteNetworkInterface","ec2:DescribeAccountAttributes","ds:AuthorizeApplication","ds:DescribeDirectories","ds:GetDirectoryLimits","ds:ListAuthorizedApplications","ds:UnauthorizeApplication","logs:DescribeLogStreams","logs:CreateLogStream","logs:PutLogEvents","logs:DescribeLogGroups","acm:GetCertificate","acm:DescribeCertificate"],"Resource":"*"}]}`),
		SetAsDefault:   true,
	})
	pVersions.Add(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String("arn:aws:iam::012345678912:policy/aws-service-role/ClientVPNServiceRolePolicy"),
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ec2:CreateNetworkInterface","ec2:CreateNetworkInterfacePermission","ec2:DescribeSecurityGroups","ec2:DescribeVpcs","ec2:DescribeSubnets","ec2:DescribeInternetGateways","ec2:ModifyNetworkInterfaceAttribute","ec2:DeleteNetworkInterface","ec2:DescribeAccountAttributes","ds:AuthorizeApplication","ds:DescribeDirectories","ds:GetDirectoryLimits","ds:UnauthorizeApplication","logs:DescribeLogStreams","logs:CreateLogStream","logs:PutLogEvents","logs:DescribeLogGroups","acm:GetCertificate","acm:DescribeCertificate"],"Resource":"*"}]}`),
		SetAsDefault:   true,
	})
	pVersions.Add(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String("arn:aws:iam::012345678912:policy/aws-service-role/ClientVPNServiceRolePolicy"),
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ec2:CreateNetworkInterface","ec2:CreateNetworkInterfacePermission","ec2:DescribeSecurityGroups","ec2:DescribeVpcs","ec2:DescribeSubnets","ec2:DescribeInternetGateways","ec2:ModifyNetworkInterfaceAttribute","ec2:DeleteNetworkInterface","ec2:DescribeAccountAttributes","ds:AuthorizeApplication","ds:DescribeDirectories","ds:GetDirectoryLimits","ds:UnauthorizeApplication","logs:DescribeLogStreams","logs:CreateLogStream","logs:PutLogEvents","logs:DescribeLogGroups","acm:GetCertificate","acm:DescribeCertificate","iam:GetSAMLProvider"],"Resource":"*"}]}`),
		SetAsDefault:   true,
	})
	pVersions.Add(&iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String("arn:aws:iam::012345678912:policy/aws-service-role/ClientVPNServiceRolePolicy"),
		PolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ec2:CreateNetworkInterface","ec2:CreateNetworkInterfacePermission","ec2:DescribeSecurityGroups","ec2:DescribeVpcs","ec2:DescribeSubnets","ec2:DescribeInternetGateways","ec2:ModifyNetworkInterfaceAttribute","ec2:DeleteNetworkInterface","ec2:DescribeAccountAttributes","ds:AuthorizeApplication","ds:DescribeDirectories","ds:GetDirectoryLimits","ds:UnauthorizeApplication","logs:DescribeLogStreams","logs:CreateLogStream","logs:PutLogEvents","logs:DescribeLogGroups","acm:GetCertificate","acm:DescribeCertificate","iam:GetSAMLProvider","lambda:GetFunctionConfiguration"],"Resource":"*"}]}`),
		SetAsDefault:   true,
	})
	v.cache.Store("ClientVPNServiceRolePolicy", pVersions)
}
