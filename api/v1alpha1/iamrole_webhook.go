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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var iamrolelog = logf.Log.WithName("iamrole-resource")

func (r *IamRole) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aws-jackhoman-com-v1alpha1-iamrole,mutating=true,failurePolicy=fail,sideEffects=None,groups=aws.jackhoman.com,resources=iamroles,verbs=create;update,versions=v1alpha1,name=miamrole.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &IamRole{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *IamRole) Default() {
	iamrolelog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:path=/validate-aws-jackhoman-com-v1alpha1-iamrole,mutating=false,failurePolicy=fail,sideEffects=None,groups=aws.jackhoman.com,resources=iamroles,verbs=create;update,versions=v1alpha1,name=viamrole.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &IamRole{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *IamRole) ValidateCreate() error {
	iamrolelog.Info("validate create", "name", r.Name)

	var errs field.ErrorList

	if r.Spec.ServiceAccounts == nil || len(r.Spec.ServiceAccounts) == 0 {
		errs = append(errs, field.Invalid(
			field.NewPath("spec").Child("serviceAccounts"),
			r.Spec.ServiceAccounts,
			"Must name service accounts to establish upstream iam trust policy document",
		))
	}
	if len(errs) == 0 {
		return nil
	}
	grp := schema.GroupKind{Group: GroupVersion.Group, Kind: r.Kind}
	return apierrors.NewInvalid(grp, r.GetName(), errs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *IamRole) ValidateUpdate(old runtime.Object) error {
	iamrolelog.Info("validate update", "name", r.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *IamRole) ValidateDelete() error {
	iamrolelog.Info("validate delete", "name", r.Name)
	return nil
}
