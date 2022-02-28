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


const (
	PolicyEffectAllow = "Allow"
	PolicyEffectDeny = "Deny"
)

type Condition struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type Conditions struct {
	ArnLike                           []Condition `json:"arnLike,omitempty"`
	ArnLikeIfExists                   []Condition `json:"arnLikeIfExists,omitempty"`
	ArnNotLike                        []Condition `json:"arnNotLike,omitempty"`
	ArnNotLikeIfExists                []Condition `json:"arnNotLikeIfExists,omitempty"`
	BinaryEquals                      []Condition `json:"binaryEquals,omitempty"`
	BinaryEqualsIfExists              []Condition `json:"binaryEqualsIfExists,omitempty"`
	Bool                              []Condition `json:"bool,omitempty"`
	BoolIfExists                      []Condition `json:"boolIfExists,omitempty"`
	DateEquals                        []Condition `json:"dateEquals,omitempty"`
	DateEqualsIfExists                []Condition `json:"dateEqualsIfExists,omitempty"`
	DateNotEquals                     []Condition `json:"dateNotEquals,omitempty"`
	DateNotEqualsIfExists             []Condition `json:"dateNotEqualsIfExists,omitempty"`
	DateLessThan                      []Condition `json:"dateLessThan,omitempty"`
	DateLessThanIfExists              []Condition `json:"dateLessThanIfExists,omitempty"`
	DateLessThanEquals                []Condition `json:"dateLessThanEquals,omitempty"`
	DateLessThanEqualsIfExists        []Condition `json:"dateLessThanEqualsIfExists,omitempty"`
	DateGreaterThan                   []Condition `json:"dateGreaterThan,omitempty"`
	DateGreaterThanIfExists           []Condition `json:"dateGreaterThanIfExists,omitempty"`
	DateGreaterThanEquals             []Condition `json:"dateGreaterThanEquals,omitempty"`
	DateGreaterThanEqualsIfExists     []Condition `json:"dateGreaterThanEqualsIfExists,omitempty"`
	IpAddress                         []Condition `json:"ipAddress,omitempty"`
	IpAddressIfExists                 []Condition `json:"ipAddressIfExists,omitempty"`
	NotIpAddress                      []Condition `json:"notIpAddress,omitempty"`
	NotIpAddressIfExists              []Condition `json:"notIpAddressIfExists,omitempty"`
	NumericEquals                     []Condition `json:"numericEquals,omitempty"`
	NumericEqualsIfExists             []Condition `json:"numericEqualsIfExists,omitempty"`
	NumericNotEquals                  []Condition `json:"numericNotEquals,omitempty"`
	NumericNotEqualsIfExists          []Condition `json:"numericNotEqualsIfExists,omitempty"`
	NumericLessThan                   []Condition `json:"numericLessThan,omitempty"`
	NumericLessThanIfExists           []Condition `json:"numericLessThanIfExists,omitempty"`
	NumericLessThanEquals             []Condition `json:"numericLessThanEquals,omitempty"`
	NumericLessThanEqualsIfExists     []Condition `json:"numericLessThanEqualsIfExists,omitempty"`
	NumericGreaterThan                []Condition `json:"numericGreaterThan,omitempty"`
	NumericGreaterThanIfExists        []Condition `json:"numericGreaterThanIfExists,omitempty"`
	NumericGreaterThanEquals          []Condition `json:"numericGreaterThanEquals,omitempty"`
	NumericGreaterThanEqualsIfExists  []Condition `json:"numericGreaterThanEqualsIfExists,omitempty"`
	Null                              []Condition `json:"null,omitempty"`
	StringLike                        []Condition `json:"stringLike,omitempty"`
	StringLikeIfExists                []Condition `json:"stringLikeIfExists,omitempty"`
	StringNotLike                     []Condition `json:"stringNotLike,omitempty"`
	StringNotLikeIfExists             []Condition `json:"stringNotLikeIfExists,omitempty"`
	StringEquals                      []Condition `json:"stringEquals,omitempty"`
	StringEqualsIfExists              []Condition `json:"stringEqualsIfExists,omitempty"`
	StringNotEquals                   []Condition `json:"stringNotEquals,omitempty"`
	StringNotEqualsIfExists           []Condition `json:"stringNotEqualsIfExists,omitempty"`
	StringEqualsIgnoreCase            []Condition `json:"stringEqualsIgnoreCase,omitempty"`
	StringEqualsIgnoreCaseIfExists    []Condition `json:"stringEqualsIgnoreCaseIfExists,omitempty"`
	StringNotEqualsIgnoreCase         []Condition `json:"stringNotEqualsIgnoreCase,omitempty"`
	StringNotEqualsIgnoreCaseIfExists []Condition `json:"stringNotEqualsIgnoreCaseIfExists,omitempty"`
}

type Statement struct {
	Sid        string      `json:"sid,omitempty"`
	// +kubebuilder:validation:Enum=Allow;Deny
	Effect     string      `json:"effect"`
	Actions    []string    `json:"action,omitempty"` // this can also be a string
	Resources  []string    `json:"resource"`
	Conditions *Conditions `json:"Condition,omitempty"`
}

type IamPolicyDocument struct {
	Version    string      `json:"version,omitempty"`
	Statements []Statement `json:"statement"`
}

// IamPolicySpec defines the desired state of IamPolicy
type IamPolicySpec struct {
	// Document - Iam policy document
	Document IamPolicyDocument `json:"document"`
}

// IamPolicyStatus defines the observed state of IamPolicy
type IamPolicyStatus struct {
	Md5Sum string `json:"md5,omitempty"`
	Arn    string `json:"arn,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// IamPolicy is the Schema for the iampolicies API
type IamPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IamPolicySpec   `json:"spec,omitempty"`
	Status IamPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IamPolicyList contains a list of IamPolicy
type IamPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IamPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IamPolicy{}, &IamPolicyList{})
}
