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
	"errors"
	"fmt"
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"strings"
)

/*
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::your-account-id:oidc-provider/oidc.eks.your-region-code.amazonaws.com/id/EXAMPLE_OIDC_IDENTIFIER"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.your-region-code.amazonaws.com/id/EXAMPLE_OIDC_IDENTIFIER:sub": "system:serviceaccount:your-namespace:your-service-account"
        }
      }
    }
  ]
}
*/

// ToPolicyDocument takes an iamrole.aws.jackhoman.com/v1alpha1 and creates a trust policy
// from the spec. The EKS cluster oidc provider arn is also required for creating the policy
func ToPolicyDocument(instance *v1alpha1.IamRole, oidcProviderArn string) (PolicyDocument, error) {
	doc := PolicyDocument{
		Version:   "2012-10-17",
		Statement: make([]StatementEntry, 0),
	}

	ix := strings.Index(oidcProviderArn, "/")
	if ix < 0 {
		return doc, errors.New("unexpected value provided for oidc provider arn")
	}
	issuerUrl := fmt.Sprintf("%s:sub", oidcProviderArn[ix+1:])
	serviceAccounts := make([]interface{}, 0)

	for _, sa := range instance.Spec.ServiceAccounts {
		serviceAccounts = append(
			serviceAccounts,
			fmt.Sprintf("system:serviceaccount:%s:%s", instance.GetNamespace(), sa.Name),
		)
	}
	var entry interface{}
	if len(serviceAccounts) == 1 {
		entry = serviceAccounts[0].(interface{})
	} else {
		entry = serviceAccounts
	}

	st := StatementEntry{
		Effect:    "Allow",
		Principal: map[string]interface{}{"Federated": oidcProviderArn},
		Action:    "sts:AssumeRoleWithWebIdentity",
		Condition: map[string]interface{}{"StringEquals": map[string]interface{}{issuerUrl: entry}},
	}
	doc.Statement = append(doc.Statement, st)
	return doc, nil
}
