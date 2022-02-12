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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IamRoleBindingSpec struct {
	IamRoleRef        string `json:"iamRoleRef"`
	ServiceAccountRef string `json:"serviceAccountRef"`
}

type IamRoleBindingStatus struct {
	BoundServiceAccountRef string `json:"serviceAccount,omitempty"`
	BoundIamRoleArn        string `json:"iamRoleArn,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.iamRoleRef`
//+kubebuilder:printcolumn:name="ServiceAccount",type=string,JSONPath=`.spec.serviceAccountRef`

// IamRoleBinding is the Schema for the iamrolebindings API
type IamRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IamRoleBindingSpec   `json:"spec,omitempty"`
	Status IamRoleBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IamRoleBindingList contains a list of IamRoleBinding
type IamRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IamRoleBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IamRoleBinding{}, &IamRoleBindingList{})
}
