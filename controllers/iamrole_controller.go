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
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"github.com/johnhoman/aws-iam-controller/pkg/bindmanager"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	notify        Notifier
	RoleService   iamrole.Interface
	DefaultPolicy string
	bindmanager.Manager
}

//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings,verbs=get;list;watch;
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *IamRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	k8 := client.NewNamespacedClient(r.Client, req.Namespace)

	instance := &v1alpha1.IamRole{}
	if err := k8.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !instance.DeletionTimestamp.IsZero() {
		// Delete resources
		if err := r.Finalize(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}

		if cu.ContainsFinalizer(instance, Finalizer) {
			// Need to establish ownership above to remove this finalizer if it somehow
			// already exists on the object
			if err := r.RemoveFinalizer(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	// If this finalizer already exists on the object but
	// isn't owned by this controller this prevents it from being owned
	// TODO: fix this
	if !cu.ContainsFinalizer(instance, Finalizer) {
		if err := r.AddFinalizer(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger = logger.WithValues("RoleName", instance.GetName())
	logger.Info("reconciling iam role")
	upstream := &iamrole.IamRole{}
	out, err := r.RoleService.Get(ctx, &iamrole.GetOptions{Name: instance.GetName()})
	if err != nil {
		if !pkgaws.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		out, err := r.CreateIamRole(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		r.notify.Created(out.Name)
		*upstream = *out
	} else {
		*upstream = *out
	}
	old := instance.DeepCopy()
	instance.Status.RoleArn = upstream.Arn
	if !reflect.DeepEqual(old.Status, instance.Status) {
		patch := client.MergeFrom(old)
		if err := k8.Status().Patch(ctx, instance, patch); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *IamRoleReconciler) UpdateTrustPolicy(ctx context.Context, instance *v1alpha1.IamRole) error {
	k8s := client.NewNamespacedClient(r.Client, instance.GetNamespace())
	logger := log.FromContext(ctx).WithValues("method", "UpdateTrustPolicy")
	logger.Info("updating trust policy for iam role")

	bindings := &v1alpha1.IamRoleBindingList{}
	if err := k8s.List(ctx, bindings, client.MatchingFields{"spec.iamRoleRef": instance.GetName()}); err != nil {
		return err
	}
	names := make([]string, len(bindings.Items))
	for _, binding := range bindings.Items {
		names = append(names, binding.Spec.ServiceAccountRef)
	}
	binding := bindmanager.Binding{Role: instance, ServiceAccounts: names}
	if err := r.Bind(ctx, &binding); err != nil {
		return err
	}

	return nil
}

func (r *IamRoleReconciler) CreateIamRole(ctx context.Context, instance *v1alpha1.IamRole) (*iamrole.IamRole, error) {
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

func (r *IamRoleReconciler) AddFinalizer(ctx context.Context, instance *v1alpha1.IamRole) error {
	k8 := client.NewNamespacedClient(r.Client, instance.GetNamespace())
	logger := log.FromContext(ctx).WithValues("method", "AddFinalizer")
	patch := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": []string{Finalizer},
		},
	}}
	patch.SetName(instance.GetName())
	patch.SetGroupVersionKind(instance.GroupVersionKind())
	logger.Info("adding finalizer")
	if err := k8.Patch(ctx, patch, client.Apply, FieldOwner, client.ForceOwnership); err != nil {
		return err
	}
	return nil
}

func (r *IamRoleReconciler) RemoveFinalizer(ctx context.Context, instance *v1alpha1.IamRole) error {
	k8 := client.NewNamespacedClient(r.Client, instance.GetNamespace())
	logger := log.FromContext(ctx).WithValues("method", "RemoveFinalizer")

	patch := client.MergeFrom(instance.DeepCopy())
	cu.RemoveFinalizer(instance, Finalizer)
	if err := k8.Patch(ctx, instance, patch, FieldOwner); err != nil {
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
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.IamRoleBinding{}, "spec.iamRoleRef", func(obj client.Object) []string {
		binding, ok := obj.(*v1alpha1.IamRoleBinding)
		if !ok {
			return []string{}
		}
		return []string{binding.Spec.IamRoleRef}
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
							Name:      binding.Spec.IamRoleRef,
							Namespace: binding.GetNamespace(),
						},
					}}
				}
				return []ctrl.Request{}
			}),
		).
		Complete(r)
}
