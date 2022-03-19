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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IamRoleSpec defines the desired state of IamRole
type IamRoleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of IamRole. Edit iamrole_types.go to remove/update
	Description        string                   `json:"description,omitempty"`
	MaxDurationSeconds int                      `json:"maxDurationSeconds,omitempty"`
	ManagedPolicies    []corev1.ObjectReference `json:"policyRefs,omitempty"`
}

// IamRoleStatus defines the observed state of IamRole
type IamRoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	RoleArn              string                   `json:"arn,omitempty"`
	BoundServiceAccounts []corev1.ObjectReference `json:"boundServiceAccounts,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// IamRole is the Schema for the iamroles API
type IamRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IamRoleSpec   `json:"spec,omitempty"`
	Status IamRoleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IamRoleList contains a list of IamRole
type IamRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IamRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IamRole{}, &IamRoleList{})
}
