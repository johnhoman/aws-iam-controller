# aws-iam-controller
Kubernetes controller for IAM resources on EKS WIP


## Custom Resources


### IamRole
```yaml
apiVersion: aws.jackhoman.com/v1alpha1
kind: IamRole
metadata:
  name: webservice
  namespace: production
spec:
  description: "Iam role for the webservice application"
  maxDurationSeconds: 3600
```

### IamRoleBinding
```yaml
apiVersion: aws.jackhoman.com/v1alpha1
kind: IamRoleBinding
metadata:
  name: webservice-binding
  namespace: production
spec:
  iamRoleRef: webservice
  serviceAccountRef: webservice
```
