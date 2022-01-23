package aws_test

/*
var _ = Describe("Utils", func() {
	It("Should create a trust policy from an iam role", func() {
		instance := &v1alpha1.IamRole{}
		instance.SetName("webapp")
		instance.SetNamespace("trust-policy")
		instance.Spec.ServiceAccounts = []corev1.LocalObjectReference{
			{Name: "webapp"},
		}
		oidcProviderArn := "arn:aws:iam::012345678912:oidc-provider/oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B716D3041E"
		doc, err := aws.ToPolicyDocument(instance, oidcProviderArn)
		Expect(err).ToNot(HaveOccurred())
		expected := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Federated": oidcProviderArn,
					},
					"Action": "sts:AssumeRoleWithWebIdentity",
					"Condition": map[string]interface{}{
						"StringEquals": map[string]interface{}{
							"oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B716D3041E:sub": "system:serviceaccount:trust-policy:webapp",
						},
					},
				},
			},
		}
		raw, err := json.Marshal(doc)
		Expect(err).ToNot(HaveOccurred())
		var m map[string]interface{}
		Expect(json.Unmarshal(raw, &m)).To(Succeed())
		Expect(m).To(Equal(expected))
	})
})
*/
