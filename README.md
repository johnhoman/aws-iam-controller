# aws-iam-controller
Kubernetes controller for IAM resources on EKS WIP


## Install

### Install [cert-manager]
The webhook configurations require encryption cert-manager will need to be installed
to run this controller

```shell
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.7.0/cert-manager.yaml
```

The webhook server uses self-signed certificates (via lets-encrypt) that the Kubernetes API service
trusts because the CA is injected into the webhook configurations


### Create IAM role for aws iam controller

The Terraform module in the repo root has the iam role and policy config required to run
the controller

Reference the config in a new Terraform file
```terraform
module "aws-iam-controller" {
  source = "github.com/johnhoman/aws-iam-controller//terraform?=main"
  
  account_id = "0123456789012"
  oidc_issuer = "oidc.eks.region-code.amazonaws.com/id/EXAMPLED539D4633E53DE1B716D3041E"
}
output "role-arn" {
  value = module.aws-iam-controller.role-arn
}
```

or just clone the repo and run locally

```shell
git clone https://github.com/johnhoman/aws-iam-controller
cd aws-iam-controller/terraform
terraform init
terraform apply -auto-approve -var=accound_id=09123456789012 -var=oidc_issuer="oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B716D3041E"
# get the role-arn
AWS_ROLE_ARN=$(terraform output role-arn)
```

### Install the controller
```shell
kubectl apply -k "github.com/johnhoman/aws-iam-controller/config/default?ref=main"
```

### Patch the service account
```shell
kubectl annotate serviceaccount aws-iam-controller-manager \
    eks.amazonaws.com/role-arn=${AWS_ROLE_ARN} \
    -n aws-iam-controller-system
```

### Patch the deployment
```shell
cat <<EOF > patch.json
[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/1/args/-",
    "value": "--oidc-arn=arn:aws:iam::<account-id>:oidc-provider/<oidc-issuer>"
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/1/args/-",
    "value": "--resource-default-path=<cluster-name>"
  }
]
EOF
kubectl patch deployment aws-iam-controller-controller-manager \
    --namespace aws-iam-controller-system \
    --type='json' \
    --patch "$(cat patch.json)"
```


## Custom Resources

### IamRole
An IamRole is a cluster scoped resource
```yaml
apiVersion: aws.jackhoman.com/v1alpha1
kind: IamRole
metadata:
  name: webservice
spec:
  description: "Iam role for the webservice application"
  maxDurationSeconds: 3600
  policyRefs:
  - name: webservice
```

### IamRoleBinding
An IamRoleBinding is namespace scoped and supports binding
roles to service accounts within the same namespace

```yaml
apiVersion: aws.jackhoman.com/v1alpha1
kind: IamRoleBinding
metadata:
  name: webservice-binding
  namespace: production
spec:
  iamRoleRef:
    name: webservice
  serviceAccountRef:
    name: webservice
```

### IamPolicy

```yaml
apiVersion: aws.jackhoman.com/v1alpha1
kind: IamPolicy
metadata:
  name: webservice
spec:
  document:
    statement:
    - sid: "AllowS3Access"
      action: "Allow"
      resource:
      - "s3:*"
      condition:
        stringLike:
        - name: "ec2:InstanceType"
          values: ["t1.*", "t2.*", "m3.*"]
        
```

### Notes
~ 16 minutes to bring up and eks control plane
