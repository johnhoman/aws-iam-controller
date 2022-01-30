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
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/pkg/bindmanager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cu "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"
)

const IamRoleBindingOwnerAnnotation = "aws.jackhoman.com/iam-role-binding"

// IamRoleBindingReconciler reconciles a IamRoleBinding object
type IamRoleBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder

	mutex       sync.Mutex
	bindManager bindmanager.Manager
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
	serviceAccount.SetName(instance.Spec.ServiceAccountRef)
	serviceAccount.SetNamespace(req.Namespace)
	// This logic is a little messy -- essentially, we don't want to steal ownership from a service account
	// that is already bound to something that isn't owned by this binding
	binding := &bindmanager.Binding{Role: iamRole, ServiceAccount: serviceAccount}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	ok, err := r.bindManager.IsBound(ctx, binding)
	if err != nil {
		logger.Error(err, "unable to get bind status")
		return ctrl.Result{}, err
	}
	if !ok {
		if err := r.bindManager.Bind(ctx, binding); err != nil {
			logger.Error(err, "unable to bind role to service account")
			return ctrl.Result{}, err
		}
	}

	if err := k8s.Get(ctx, types.NamespacedName{Name: instance.Spec.ServiceAccountRef}, serviceAccount); err != nil {
		r.Event(instance, corev1.EventTypeWarning, "ServiceAccountNotFound", fmt.Sprintf(
			"ServiceAccount %s not found",
			instance.Spec.ServiceAccountRef,
		))
		return ctrl.Result{}, err
	}

	_, found := serviceAccount.GetAnnotations()[bindmanager.IamRoleArnAnnotation]
	if !found {
		// Not found then add it
		if owner, ok := serviceAccount.GetAnnotations()[IamRoleBindingOwnerAnnotation]; ok {
			if owner != instance.GetName() {
				// The service account is already owned
				r.Event(instance, corev1.EventTypeWarning, "Conflict", fmt.Sprintf(
					"ServiceAccount %s already managed by %s", serviceAccount.GetName(), owner,
				))
				return ctrl.Result{}, fmt.Errorf("service account already managed by %s", owner)
			}
		} else {
			// not found
			logger.Info("setting owner annotation")
			// Own it
			patch := client.MergeFrom(serviceAccount.DeepCopy())
			annotations := serviceAccount.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations[IamRoleBindingOwnerAnnotation] = instance.GetName()
			serviceAccount.SetAnnotations(annotations)
			// This patch isn't working for some reason
			if err := k8s.Patch(ctx, serviceAccount, patch, client.FieldOwner(instance.GetName())); err != nil {
				logger.Error(err, "failed to tag service account with owner")
				return ctrl.Result{}, err
			}
			logger.Info("patched service account with owner reference")
		}
		logger.Info("adding annotation for iam role to service account", "roleName", iamRole.GetName())
		// This is removing the owner annotation ?
		if err := r.bindManager.Patch(binding, client.FieldOwner(instance.GetName())).Do(ctx, k8s); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("patched service account", "serviceAccountName", serviceAccount.GetName(), "roleArn", iamRole.Status.RoleArn)
	}

	// the service account needs another annotation after this
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IamRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: Figure out the how to re-trigger on iamRole
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IamRoleBinding{}).
		Watches(
			&source.Kind{Type: &corev1.ServiceAccount{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []ctrl.Request {
				if obj.GetAnnotations() != nil && len(obj.GetAnnotations()) > 0 {
					binding, ok := obj.GetAnnotations()[IamRoleBindingOwnerAnnotation]
					if ok {
						return []ctrl.Request{{
							NamespacedName: types.NamespacedName{
								Name:      binding,
								Namespace: obj.GetNamespace(),
							},
						}}
					}
				}
				return []ctrl.Request{}
			}),
		).
		Complete(r)
}
