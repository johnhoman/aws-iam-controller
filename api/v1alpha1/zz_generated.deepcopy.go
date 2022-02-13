//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IamRole) DeepCopyInto(out *IamRole) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IamRole.
func (in *IamRole) DeepCopy() *IamRole {
	if in == nil {
		return nil
	}
	out := new(IamRole)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IamRole) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IamRoleBinding) DeepCopyInto(out *IamRoleBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IamRoleBinding.
func (in *IamRoleBinding) DeepCopy() *IamRoleBinding {
	if in == nil {
		return nil
	}
	out := new(IamRoleBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IamRoleBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IamRoleBindingList) DeepCopyInto(out *IamRoleBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IamRoleBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IamRoleBindingList.
func (in *IamRoleBindingList) DeepCopy() *IamRoleBindingList {
	if in == nil {
		return nil
	}
	out := new(IamRoleBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IamRoleBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IamRoleBindingSpec) DeepCopyInto(out *IamRoleBindingSpec) {
	*out = *in
	out.IamRoleRef = in.IamRoleRef
	out.ServiceAccountRef = in.ServiceAccountRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IamRoleBindingSpec.
func (in *IamRoleBindingSpec) DeepCopy() *IamRoleBindingSpec {
	if in == nil {
		return nil
	}
	out := new(IamRoleBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IamRoleBindingStatus) DeepCopyInto(out *IamRoleBindingStatus) {
	*out = *in
	out.BoundServiceAccountRef = in.BoundServiceAccountRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IamRoleBindingStatus.
func (in *IamRoleBindingStatus) DeepCopy() *IamRoleBindingStatus {
	if in == nil {
		return nil
	}
	out := new(IamRoleBindingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IamRoleList) DeepCopyInto(out *IamRoleList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IamRole, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IamRoleList.
func (in *IamRoleList) DeepCopy() *IamRoleList {
	if in == nil {
		return nil
	}
	out := new(IamRoleList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IamRoleList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IamRoleSpec) DeepCopyInto(out *IamRoleSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IamRoleSpec.
func (in *IamRoleSpec) DeepCopy() *IamRoleSpec {
	if in == nil {
		return nil
	}
	out := new(IamRoleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IamRoleStatus) DeepCopyInto(out *IamRoleStatus) {
	*out = *in
	if in.BoundServiceAccounts != nil {
		in, out := &in.BoundServiceAccounts, &out.BoundServiceAccounts
		*out = make([]v1.ObjectReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IamRoleStatus.
func (in *IamRoleStatus) DeepCopy() *IamRoleStatus {
	if in == nil {
		return nil
	}
	out := new(IamRoleStatus)
	in.DeepCopyInto(out)
	return out
}
