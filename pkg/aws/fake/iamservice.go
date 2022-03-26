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
	"math/rand"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
)

type Cache struct {
	Attachments     sync.Map
	Roles           sync.Map
	ManagedPolicies sync.Map
	// mapping ARNs to policy names
	policyArnMapping sync.Map
}

type IamService struct {
	AccountID string
	Cache
}

type policyVersions struct {
	sync.Map
}

func (pv *policyVersions) Len() int {
	count := 0
	pv.Range(func(_ interface{}, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (pv *policyVersions) Default() iamtypes.PolicyVersion {
	v := &iamtypes.PolicyVersion{}
	pv.Range(func(key interface{}, value interface{}) bool {
		p := value.(iamtypes.PolicyVersion)
		if p.IsDefaultVersion {
			*v = p
		}
		return false
	})
	return *v
}

func (pv *policyVersions) Latest() iamtypes.PolicyVersion {
	policy := &iamtypes.PolicyVersion{
		CreateDate: aws.Time(time.Now().UTC().Add(time.Second * 3600)),
	}
	pv.Range(func(key interface{}, value interface{}) bool {
		v := value.(iamtypes.PolicyVersion)
		if aws.ToTime(v.CreateDate).Before(aws.ToTime(policy.CreateDate)) {
			*policy = v
		}
		return true
	})
	return *policy
}

type managedPolicy struct {
	versions *policyVersions
	policy   iamtypes.Policy
}

func (i *IamService) Reset() {
	i.Cache = Cache{
		Attachments:      sync.Map{},
		Roles:            sync.Map{},
		ManagedPolicies:  sync.Map{},
		policyArnMapping: sync.Map{},
	}

	i.Roles.Store("AWSServiceRoleForAmazonEKS", &iamtypes.Role{
		Arn:                      aws.String("arn:aws:iam::0123456789012:role/aws-service-role/eks.amazonaws.com/AWSServiceRoleForAmazonEKS"),
		CreateDate:               aws.Time(time.Date(2020, 5, 10, 3, 47, 44, 0, time.UTC)),
		Path:                     aws.String("/aws-service-role/eks.amazonaws.com/"),
		RoleId:                   aws.String("AROBTHUTBNP8XYHIZQB7U"),
		RoleName:                 aws.String("AWSServiceRoleForAmazonEKS"),
		AssumeRolePolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"eks.amazonaws.com"},"Action":"sts:AssumeRole"}]}`),
		Description:              aws.String("Allows Amazon EKS to call AWS services on your behalf."),
		MaxSessionDuration:       aws.Int32(3600),
		RoleLastUsed: &iamtypes.RoleLastUsed{
			LastUsedDate: aws.Time(time.Date(2022, 2, 8, 3, 19, 36, 0, time.UTC)),
			Region:       aws.String("us-east-1"),
		},
	})
	mp := managedPolicy{
		policy: iamtypes.Policy{
			Arn:              aws.String("arn:aws:iam::aws:policy/AmazonEC2FullAccess"),
			AttachmentCount:  aws.Int32(0),
			CreateDate:       aws.Time(time.Date(2015, 2, 6, 18, 40, 15, 0, time.UTC)),
			DefaultVersionId: aws.String("v5"),
			Description:      aws.String(" Provides full access to Amazon EC2 via the AWS Management Console. "),
			IsAttachable:     true,
			Path:             aws.String("/"),
			PolicyId:         aws.String("ANPAI3VAJF5ZCRZ7MCQE6"),
			PolicyName:       aws.String("AmazonEC2FullAccess"),
		},
	}
	mp.versions = &policyVersions{Map: sync.Map{}}
	mp.versions.Store("v1", iamtypes.PolicyVersion{
		Document:         aws.String(`{"Version":"2012-10-17","Statement":[{"Action":"ec2:*","Effect":"Allow","Resource":"*"},{"Effect":"Allow","Action":"elasticloadbalancing:*","Resource":"*"},{"Effect":"Allow","Action":"cloudwatch:*","Resource":"*"},{"Effect":"Allow","Action":"autoscaling:*","Resource":"*"},{"Effect":"Allow","Action":"iam:CreateServiceLinkedRole","Resource":"*","Condition":{"StringEquals":{"iam:AWSServiceName":["autoscaling.amazonaws.com","ec2scheduled.amazonaws.com","elasticloadbalancing.amazonaws.com","spot.amazonaws.com","spotfleet.amazonaws.com","transitgateway.amazonaws.com"]}}}]}`),
		VersionId:        aws.String("v5"),
		IsDefaultVersion: true,
		CreateDate:       aws.Time(time.Date(2018, 11, 27, 2, 16, 56, 0, time.UTC)),
	})
	i.policyArnMapping.Store("arn:aws:iam::aws:policy/AmazonEC2FullAccess", "AmazonEC2FullAccess")
}

var _ pkgaws.IamService = &IamService{}

func NewIamService() *IamService {
	i := &IamService{AccountID: "012345678912"}
	i.Reset()
	return i
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
