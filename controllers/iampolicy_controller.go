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
	"crypto/md5"
	"fmt"

	awsv1alpha1 "github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iampolicy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IamPolicyReconciler reconciles a IamPolicy object
type IamPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder

	AWS iampolicy.Interface
}

const (
	IamPolicyFinalizer  = "aws.jackhoman.com/delete-iam-policy"
	IamPolicyFieldOwner = client.FieldOwner("policy.iam.aws.controller")
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
			if len(instance.Status.Arn) != 0 {
				if err := r.AWS.Delete(ctx, &iampolicy.DeleteOptions{Arn: instance.Status.Arn}); err != nil {
					logger.Error(err, "unable to delete iam policy", "arn", instance.Status.Arn)
					return ctrl.Result{}, err
				}
				logger.Info("deleted resource", "arn", instance.Status.Arn)
			} else {
				logger.Info("arn not found in status")
			}

			patch := client.MergeFrom(instance.DeepCopy())
			controllerutil.RemoveFinalizer(instance, IamPolicyFinalizer)
			if err := r.Client.Patch(ctx, instance, patch); err != nil {
				logger.Error(err, "unable to remove finalizer")
				return ctrl.Result{}, err
			}
			logger.Info("removed finalizer")
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
	// Create the iam policy
	options := &iampolicy.GetOptions{Name: instance.GetName()}
	if len(instance.Status.Arn) > 0 {
		// Use the arn if it's available. Most of the time it should be.
		// Using the name to get the arn will be a more expensive operation
		*options = iampolicy.GetOptions{Arn: instance.Status.Arn}
	}
	document, err := serializeDocument(instance)
	iamPolicy, err := r.AWS.Get(ctx, options)
	sum := md5Sum(document)
	if err != nil {
		if !aws.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		// Create it
		iamPolicy, err = r.AWS.Create(ctx, &iampolicy.CreateOptions{
			Name:        instance.GetName(),
			Document:    document,
			Description: instance.Spec.Description,
		})
		if err != nil {
			logger.Error(err, "unable to create iam policy")
			return ctrl.Result{}, err
		}
		patch := &unstructured.Unstructured{Object: map[string]interface{}{
			"status": map[string]interface{}{
				"arn": iamPolicy.Arn,
				"md5": sum,
			},
		}}
		patch.SetGroupVersionKind(instance.GroupVersionKind())
		patch.SetName(instance.GetName())
		if err := r.Client.Status().Patch(ctx, patch, client.Apply, IamPolicyFieldOwner, client.ForceOwnership); err != nil {
			logger.Error(err, "unable to patch status on create")
		}
		r.Eventf(instance, corev1.EventTypeNormal, "Created", "Created iam policy %s", iamPolicy.Arn)
	}
	if sum != instance.Status.Md5Sum {
		iamPolicy, err = r.AWS.Update(ctx, &iampolicy.UpdateOptions{
			Arn:      iamPolicy.Arn,
			Document: document,
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		patch := &unstructured.Unstructured{Object: map[string]interface{}{
			"status": map[string]interface{}{
				"arn": iamPolicy.Arn,
				"md5": sum,
			},
		}}
		patch.SetGroupVersionKind(instance.GroupVersionKind())
		patch.SetName(instance.GetName())
		if err := r.Status().Patch(ctx, patch, client.Apply, IamPolicyFieldOwner, client.ForceOwnership); err != nil {
			logger.Error(err, "unable to update status after policy document update")
			return ctrl.Result{}, err
		}
		r.Eventf(instance, corev1.EventTypeNormal, "Updated", "Updated iam policy %s", iamPolicy.Arn)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IamPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.IamPolicy{}).
		Complete(r)
}

func serializeDocument(instance *awsv1alpha1.IamPolicy) (string, error) {
	// Create it
	doc := iampolicy.NewDocument()
	// TODO: use version
	statements := make([]iampolicy.Statement, 0, len(instance.Spec.Document.Statements))
	for _, statement := range instance.Spec.Document.Statements {
		statements = append(statements, iampolicy.Statement{
			Sid:      statement.Sid,
			Effect:   statement.Effect,
			Action:   statement.Actions,
			Resource: statement.Resources,
			// TODO: Conditions
		})
	}
	doc.SetStatements(statements)
	document, err := doc.Marshal()
	if err != nil {
		return "", err
	}
	return document, nil
}

func md5Sum(s string) string {
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum)
}
