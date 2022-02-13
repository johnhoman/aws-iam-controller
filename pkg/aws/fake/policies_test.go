package fake_test

import (
    "sync"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/iam"
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
        PolicyDocument:aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ec2:CreateNetworkInterface","ec2:CreateNetworkInterfacePermission","ec2:DescribeSecurityGroups","ec2:DescribeVpcs","ec2:DescribeSubnets","ec2:DescribeInternetGateways","ec2:ModifyNetworkInterfaceAttribute","ec2:DeleteNetworkInterface","ec2:DescribeAccountAttributes","ds:AuthorizeApplication","ds:DescribeDirectories","ds:GetDirectoryLimits","ds:UnauthorizeApplication","logs:DescribeLogStreams","logs:CreateLogStream","logs:PutLogEvents","logs:DescribeLogGroups","acm:GetCertificate","acm:DescribeCertificate","iam:GetSAMLProvider","lambda:GetFunctionConfiguration"],"Resource":"*"}]}`),
        PolicyName: aws.String("ClientVPNServiceRolePolicy"),
        Description: aws.String(" Policy to enable AWS Client VPN to manage your Client VPN endpoints."),
        Path: aws.String("/aws-service-role/"),
    })

    p.cache.Store("AWSHealthFullAccess", iam.CreatePolicyInput{
        PolicyDocument:aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["organizations:EnableAWSServiceAccess","organizations:DisableAWSServiceAccess"],"Resource":"*","Condition":{"StringEquals":{"organizations:ServicePrincipal":"health.amazonaws.com"}}},{"Effect":"Allow","Action":["health:*","organizations:ListAccounts","organizations:ListParents","organizations:DescribeAccount","organizations:ListDelegatedAdministrators"],"Resource":"*"},{"Effect":"Allow","Action":"iam:CreateServiceLinkedRole","Resource":"*","Condition":{"StringEquals":{"iam:AWSServiceName":"health.amazonaws.com"}}}]}`),
        PolicyName: aws.String("AWSHealthFullAccess"),
        Description: aws.String(" Allows full access to the AWS Health Apis and Notifications and the Personal Health Dashboard"),
    })
}