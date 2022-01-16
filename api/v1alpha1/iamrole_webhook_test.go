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

package v1alpha1_test

import (
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IamRoleWebhook", func() {
	It("Should deny a request without service accounts", func() {
		instance := &v1alpha1.IamRole{}
		instance.SetName("should-deny-no-service-account")
		Expect(k8sClient.Create(ctx, instance)).ShouldNot(Succeed())
	})
	It("Should allow a request with service accounts", func() {
		instance := &v1alpha1.IamRole{
			Spec: v1alpha1.IamRoleSpec{
				ServiceAccounts: []corev1.LocalObjectReference{{Name: "default"}},
			},
		}
		instance.SetName("should-succeed-with-service-account")
		Expect(namespacedClient.Create(ctx, instance)).Should(Succeed())
	})
})
