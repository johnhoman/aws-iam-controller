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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	awsv1alpha1 "github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"
)

// IamPolicyReconciler reconciles a IamPolicy object
type IamPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder

	AWS iampolicy.Interface
}

const (
	IamPolicyFinalizer = "aws.jackhoman.com/delete-iam-policy"
)

//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iampolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iampolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iampolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IamPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	instance := &awsv1alpha1.IamPolicy{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		logger.Info("unable to get instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !instance.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(instance, IamPolicyFinalizer) {
			// Remove the Iam Policy
			// - Check the status for an ARN

			patch := client.MergeFrom(instance)
			controllerutil.RemoveFinalizer(instance, IamPolicyFinalizer)
			if err := r.Client.Patch(ctx, instance, patch); err != nil {
				logger.Error(err, "unable to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(instance, IamPolicyFinalizer) {
		patch := &unstructured.Unstructured{Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"finalizers": []string{IamPolicyFinalizer},
			},
		}}
		patch.SetName(instance.GetName())
		patch.SetGroupVersionKind(instance.GroupVersionKind())
		if err := r.Client.Patch(ctx, patch, client.Apply, client.FieldOwner("aws-iam-policy-controller")); err != nil {
			logger.Error(err, "unable to patch finalizer", "finalizer", IamPolicyFinalizer)
			logger.Info("finalizer not added")
			return ctrl.Result{}, err
		}
		logger.Info("added finalizer", "finalizer", IamPolicyFinalizer)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IamPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.IamPolicy{}).
		Complete(r)
}
