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
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"github.com/johnhoman/aws-iam-controller/pkg/bindmanager"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cu "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	PrometheusNamespace                   = "aws_iam_controller"
	PrometheusSubsystem                   = "role_reconciler"
	Finalizer                             = "jackhoman.com/delete-iam-role"
	FieldOwner          client.FieldOwner = "aws-iam-controller"
)

var (
	upstreamPolicyDocumentInvalid = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: PrometheusNamespace,
		Subsystem: PrometheusSubsystem,
		Name:      "role_upstream_policy_document_invalid",
		Help:      "The policy document retrieved from aws for this role is invalid",
	}, []string{"roleName"})
	roleCreated = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: PrometheusNamespace,
		Subsystem: PrometheusSubsystem,
		Name:      "role_created",
		Help:      "Created a new aws iam role",
	}, []string{"roleName"})
	roleUpdated = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: PrometheusNamespace,
		Subsystem: PrometheusSubsystem,
		Name:      "role_updated",
		Help:      "Updated the aws iam role",
	}, []string{"roleName"})
	roleDeleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: PrometheusNamespace,
		Subsystem: PrometheusSubsystem,
		Name:      "role_deleted",
		Help:      "Deleted an existing aws iam role",
	}, []string{"roleName"})
	roleTrustPolicyUpdated = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: PrometheusNamespace,
		Subsystem: PrometheusSubsystem,
		Name:      "role_trust_policy_updated",
		Help:      "Deleted an existing aws iam role",
	}, []string{"roleName"})
)

func init() {
	prometheus.MustRegister(upstreamPolicyDocumentInvalid)
	prometheus.MustRegister(roleCreated)
	prometheus.MustRegister(roleUpdated)
	prometheus.MustRegister(roleDeleted)
	prometheus.MustRegister(roleTrustPolicyUpdated)
}

type Notifier interface {
	InvalidPolicyDocument(roleName string)
	Created(roleName string)
	Updated(roleName string)
	Deleted(roleName string)
	TrustPolicyUpdated(roleName string)
}

type notifier struct{}

func (n *notifier) TrustPolicyUpdated(roleName string) {
	roleTrustPolicyUpdated.WithLabelValues(roleName).Inc()
}

func (n *notifier) Created(roleName string) {
	roleCreated.WithLabelValues(roleName).Inc()
}

func (n *notifier) Updated(roleName string) {
	roleUpdated.WithLabelValues(roleName).Inc()
}

func (n *notifier) Deleted(roleName string) {
	roleDeleted.WithLabelValues(roleName).Inc()
}

func (n *notifier) InvalidPolicyDocument(roleName string) {
	upstreamPolicyDocumentInvalid.WithLabelValues(roleName).Inc()
}

var _ Notifier = &notifier{}

// IamRoleReconciler reconciles a IamRole object
type IamRoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder

	notify        Notifier
	RoleService   iamrole.Interface
	DefaultPolicy string
	bindmanager.Manager
}

//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings,verbs=get;list;watch;
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *IamRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	instance := &v1alpha1.IamRole{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		logger.Error(err, "unable to get instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !instance.DeletionTimestamp.IsZero() {
		logger.Info("instance pending deletion")
		// Delete resources
		if err := r.Finalize(ctx, instance); err != nil {
			logger.Error(err, "unable to finalize instance")
			return ctrl.Result{}, err
		}

		if cu.ContainsFinalizer(instance, Finalizer) {
			// Need to establish ownership above to remove this finalizer if it somehow
			// already exists on the object
			if err := r.RemoveFinalizer(ctx, instance); err != nil {
				logger.Error(err, "an error occurring removing the finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	// If this finalizer already exists on the object but
	// isn't owned by this controller this prevents it from being owned
	// TODO: fix this
	if !cu.ContainsFinalizer(instance, Finalizer) {
		logger.Info("adding finalizer")
		if err := r.addFinalizer(ctx, instance); err != nil {
			logger.Error(err, "unable to add finalizer")
			return ctrl.Result{}, err
		}
		logger.Info("added finalizer")
	}

	logger = logger.WithValues("RoleName", instance.GetName())
	logger.Info("reconciling iam role")
	upstream := &iamrole.IamRole{}
	out, err := r.RoleService.Get(ctx, &iamrole.GetOptions{Name: instance.GetName()})
	if err != nil {
		if !pkgaws.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		logger.Info("upstream iam role not found")
		out, err := r.createIamRole(ctx, instance)
		if err != nil {
			logger.Error(err, "unable to create iam role")
			return ctrl.Result{}, err
		}
		r.notify.Created(out.Name)
		*upstream = *out
		logger.Info("created upstream iam role", "arn", upstream.Arn)
	} else {
		*upstream = *out
		logger.Info("upstream iam role exists", "arn", upstream.Arn)
	}
	for _, ref := range instance.Spec.PolicyRefs {
		policy := &v1alpha1.IamPolicy{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name}, policy); err != nil {
			logger.Error(err, fmt.Sprintf("unable to get reference policy %s", ref.Name))
			continue
		}
	}

	if instance.Status.RoleArn != upstream.Arn {
		logger.Info("Status out of sync", "have", instance.Status.RoleArn, "want", upstream.Arn)
		patch := client.MergeFrom(instance.DeepCopy())
		instance.Status.RoleArn = upstream.Arn
		if err := r.Client.Status().Patch(ctx, instance, patch); err != nil {
			logger.Error(err, "unable to update status")
			return ctrl.Result{}, err
		}
		logger.Info("Status updated")
	}
	if err := r.updateTrustPolicy(ctx, instance); err != nil {
		logger.Error(err, "unable to update trust policy")
		return ctrl.Result{}, err
	}

	logger.Info("Reconcile complete")
	return ctrl.Result{}, nil
}

func (r *IamRoleReconciler) updateTrustPolicy(ctx context.Context, instance *v1alpha1.IamRole) error {
	logger := log.FromContext(ctx).WithValues("method", "UpdateTrustPolicy")
	logger.V(4).Info("updating trust policy for iam role")

	bindings := &v1alpha1.IamRoleBindingList{}
	if err := r.Client.List(ctx, bindings, client.MatchingFields{"spec.iamRoleRef.name": instance.GetName()}); err != nil {
		logger.Error(err, "unable to list role bindings")
		return err
	}
	for _, item := range bindings.Items {
		logger.V(4).Info("identified role binding", "bindingName", item.Name)
	}
	objectRefs := make([]corev1.ObjectReference, 0, len(bindings.Items))
	for _, binding := range bindings.Items {
		objectRefs = append(objectRefs, corev1.ObjectReference{
			Name:      binding.Spec.ServiceAccountRef.Name,
			Namespace: binding.GetNamespace(),
		})
	}
	binding := bindmanager.Binding{Role: instance, ServiceAccounts: objectRefs}
	if err := r.Bind(ctx, &binding); err != nil {
		logger.Error(err, "unable to bind service account")
		return err
	}
	if !reflect.DeepEqual(instance.Status.BoundServiceAccounts, objectRefs) {
		patch := client.MergeFrom(instance.DeepCopy())
		instance.Status.BoundServiceAccounts = objectRefs
		if err := r.Client.Status().Patch(ctx, instance, patch); err != nil {
			logger.Error(err, "unable to update status")
		}
		logger.Info("updated status with role bindings")
	}

	return nil
}

func (r *IamRoleReconciler) createIamRole(ctx context.Context, instance *v1alpha1.IamRole) (*iamrole.IamRole, error) {
	out, err := r.RoleService.Create(ctx, &iamrole.CreateOptions{
		Name:               instance.GetName(),
		MaxDurationSeconds: int32(instance.Spec.MaxDurationSeconds),
		PolicyDocument:     r.DefaultPolicy,
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *IamRoleReconciler) addFinalizer(ctx context.Context, instance *v1alpha1.IamRole) error {
	logger := log.FromContext(ctx).WithValues("method", "AddFinalizer")
	patch := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": []string{Finalizer},
		},
	}}
	patch.SetName(instance.GetName())
	patch.SetGroupVersionKind(instance.GroupVersionKind())
	if err := r.Client.Patch(ctx, patch, client.Apply, FieldOwner, client.ForceOwnership); err != nil {
		logger.Error(err, "unable to add finalizer")
		return err
	}
	logger.Info("patched finalizer")
	return nil
}

func (r *IamRoleReconciler) RemoveFinalizer(ctx context.Context, instance *v1alpha1.IamRole) error {
	logger := log.FromContext(ctx).WithValues("method", "RemoveFinalizer")

	patch := client.MergeFrom(instance.DeepCopy())
	cu.RemoveFinalizer(instance, Finalizer)
	if err := r.Client.Patch(ctx, instance, patch, FieldOwner); err != nil {
		logger.Error(err, "unable to patch finalizers")
		return err
	}
	logger.Info("removed finalizer")
	return nil
}

func (r *IamRoleReconciler) Finalize(ctx context.Context, instance *v1alpha1.IamRole) error {
	logger := log.FromContext(ctx).WithValues("method", "Finalize")
	logger.Info("Removing IAM Role")

	out, err := r.RoleService.Get(ctx, &iamrole.GetOptions{Name: instance.GetName()})
	if err != nil {
		if !pkgaws.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.RoleService.Delete(ctx, &iamrole.DeleteOptions{Name: instance.GetName()}); err != nil {
			return err
		}
		r.notify.Deleted(instance.GetName())
		logger.Info("Removed upstream role", "arn", out.Arn)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IamRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.notify = &notifier{}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.IamRoleBinding{}, "spec.iamRoleRef.name", func(obj client.Object) []string {
		binding, ok := obj.(*v1alpha1.IamRoleBinding)
		if !ok {
			return []string{}
		}
		return []string{binding.Spec.IamRoleRef.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IamRole{}).
		Watches(
			&source.Kind{Type: &v1alpha1.IamRoleBinding{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []ctrl.Request {
				binding, ok := obj.(*v1alpha1.IamRoleBinding)
				if ok {
					return []ctrl.Request{{
						NamespacedName: types.NamespacedName{
							Name:      binding.Spec.IamRoleRef.Name,
							Namespace: binding.GetNamespace(),
						},
					}}
				}
				return []ctrl.Request{}
			}),
		).
		Watches(
			&source.Kind{Type: &v1alpha1.IamPolicy{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []ctrl.Request {
				status := obj.(*v1alpha1.IamPolicy).Status
				requests := make([]ctrl.Request, 0, len(status.AttachedRoles))
				for _, ref := range status.AttachedRoles {
					requests = append(requests, ctrl.Request{
						NamespacedName: types.NamespacedName{Name: ref.Name},
					})
				}
				return requests
			}),
		).
		Complete(r)
}
