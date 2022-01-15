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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cu "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	jackhomancomv1alpha1 "github.com/johnhoman/aws-iam-controller/api/v1alpha1"
)

const (
	Finalizer                    = "jackhoman.com/delete-iam-role"
	FieldOwner client.FieldOwner = "aws-iam-controller"
)

// IamRoleReconciler reconciles a IamRole object
type IamRoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=jackhoman.com,resources=iamroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=jackhoman.com,resources=iamroles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=jackhoman.com,resources=iamroles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *IamRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	instance := &jackhomancomv1alpha1.IamRole{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if instance.DeletionTimestamp.IsZero() {
		if !cu.ContainsFinalizer(instance, Finalizer) {
			patch := &unstructured.Unstructured{Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"finalizers": []string{Finalizer},
				},
			}}
			patch.SetName(instance.GetName())
			patch.SetNamespace(instance.GetNamespace())
			patch.SetGroupVersionKind(instance.GroupVersionKind())
			logger.Info("adding finalizer")
			if err := r.Client.Patch(ctx, patch, client.Apply, FieldOwner, client.ForceOwnership); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Delete resources
		logger.Info("Removing IAM Role")

		if cu.ContainsFinalizer(instance, Finalizer) {
			// how do I remove the finalizer now with a patch?
			patch := client.MergeFrom(instance.DeepCopy())
			cu.RemoveFinalizer(instance, Finalizer)
			if err := r.Client.Patch(ctx, instance, patch, FieldOwner); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IamRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jackhomancomv1alpha1.IamRole{}).
		Complete(r)
}
