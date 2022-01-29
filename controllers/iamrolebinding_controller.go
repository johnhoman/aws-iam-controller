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

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cu "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"github.com/johnhoman/aws-iam-controller/pkg/bindmanager"
)

// IamRoleBindingReconciler reconciles a IamRoleBinding object
type IamRoleBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder

	roleService iamrole.Interface
	oidcArn     string
	issuerUrl   string
}

const (
	IamRoleArnAnnotation    = "eks.amazonaws.com/role-arn"
	IamRoleLock             = "aws.jackhoman.com/iam-role-lock"
	IamRoleBindingFinalizer = "aws.jackhoman.com/free-service-account"
)

//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles,verbs=get;list;watch;
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *IamRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	k8s := client.NewNamespacedClient(r.Client, req.Namespace)
	aws := r.roleService

	instance := &v1alpha1.IamRoleBinding{}
	if err := k8s.Get(ctx, req.NamespacedName, instance); err != nil {
		// There might be other things that need to be cleaned up here
		logger.Info("instance not found")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !instance.GetDeletionTimestamp().IsZero() {

	}
	if !cu.ContainsFinalizer(instance, IamRoleBindingFinalizer) {

	}

	// Assert that the role ref is of type iam role somewhere
	iamRole := &v1alpha1.IamRole{}
	key := types.NamespacedName{Name: instance.Spec.IamRoleRef, Namespace: instance.GetNamespace()}
	if err := k8s.Get(ctx, key, iamRole); err != nil {
		r.Eventf(instance, corev1.EventTypeWarning, "RoleNotFound", "IamRole %s not found", instance.Spec.IamRoleRef)
		return ctrl.Result{}, err
	}
	if len(iamRole.Status.RoleArn) == 0 {
		return ctrl.Result{}, fmt.Errorf("role-arn not available")
	}

	serviceAccount := &corev1.ServiceAccount{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: instance.Spec.ServiceAccountRef}, serviceAccount); err != nil {
		r.Eventf(instance, corev1.EventTypeWarning, "ServiceAccountNotFound", "ServiceAccount %s not found", instance.Spec.ServiceAccountRef)
		return ctrl.Result{}, err
	}
	// LOCK
	bindingId := fmt.Sprintf("%s/%s", instance.GetNamespace(), instance.GetName())
	patch := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{IamRoleLock: bindingId},
		},
	}}
	patch.SetGroupVersionKind(iamRole.GroupVersionKind())
	patch.SetName(iamRole.GetName())
	if err := k8s.Patch(ctx, patch, client.Apply, client.FieldOwner(bindingId)); err != nil {
		r.Event(instance, corev1.EventTypeWarning, "LockAcquisitionFailure", fmt.Sprintf(
			"Failed to acquire lock on IamRole %s", iamRole.GetName(),
		))
		return ctrl.Result{}, err
	}
	logger.Info("Acquired lock", "role", iamRole.GetName())

	bm := bindmanager.New(aws, r.oidcArn)
	binding := bindmanager.Binding{Role: iamRole, ServiceAccount: serviceAccount}
	ok, err := bm.IsBound(ctx, &binding)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !ok {
		if err := bm.Bind(ctx, &binding); err != nil {
			return ctrl.Result{}, err
		}
	}
	logger.Info("adding annotation for iam role", "roleName", iamRole.GetName())
	if err := bm.Patch(&binding, client.FieldOwner(instance.GetName())).Do(ctx, k8s); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("patched service account", "serviceAccountName", serviceAccount.GetName())
	// UNLOCK
	_, found := iamRole.GetAnnotations()[IamRoleLock]
	if found {
		unpatch := client.MergeFrom(iamRole.DeepCopy())
		annotations := iamRole.GetAnnotations()
		delete(annotations, IamRoleLock)
		iamRole.SetAnnotations(annotations)
		if err := k8s.Patch(ctx, iamRole, unpatch); err != nil {
			r.Event(instance, corev1.EventTypeWarning, "LockReleaseFailure", fmt.Sprintf(
				"Failed to release lock on IamRole %s", iamRole.GetName(),
			))
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IamRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: Figure out the how to re-trigger on service account
	// TODO: Figure out the how to re-trigger on iamRole
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IamRoleBinding{}).
		Complete(r)
}
