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
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"net/url"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cu "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

	notify Notifier
	pkgaws.IamRoleService
	oidcProviderArn string
	clusterName     string
}

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
		logger.Info("Removing IAM Role")
		out, err := r.IamRoleService.GetRole(ctx, &iam.GetRoleInput{RoleName: aws.String(roleName(instance))})
		if err != nil {
			if pkgaws.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		} else {
			if _, err := r.IamRoleService.DeleteRole(ctx, &iam.DeleteRoleInput{RoleName: out.Role.RoleName}); err != nil {
				return ctrl.Result{}, err
			}
			r.notify.Deleted(aws.ToString(out.Role.RoleName))
			logger.Info("Removed upstream role", "arn", aws.ToString(out.Role.Arn))
		}

		if cu.ContainsFinalizer(instance, Finalizer) {
			// Need to establish ownership above to remove this finalizer if it somehow
			// already exists on the object
			patch := client.MergeFrom(instance.DeepCopy())
			cu.RemoveFinalizer(instance, Finalizer)
			if err := k8.Patch(ctx, instance, patch, FieldOwner); err != nil {
				logger.Error(err, "unable to patch finalizers")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	// If this finalizer already exists on the object but
	// isn't owned by this controller this prevents it from being owned
	// TODO: fix this
	if !cu.ContainsFinalizer(instance, Finalizer) {
		patch := &unstructured.Unstructured{Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"finalizers": []string{Finalizer},
			},
		}}
		patch.SetName(instance.GetName())
		patch.SetGroupVersionKind(instance.GroupVersionKind())
		logger.Info("adding finalizer")
		if err := k8.Patch(ctx, patch, client.Apply, FieldOwner, client.ForceOwnership); err != nil {
			return ctrl.Result{}, err
		}
		if err := k8.Get(ctx, req.NamespacedName, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	name := instance.GetNamespace() + "-" + instance.GetName()
	logger = logger.WithValues("RoleName", name)
	logger.Info("reconciling iam role")
	upstream := &iamtypes.Role{}
	out, err := r.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		if pkgaws.IsNotFound(err) {
			doc, err := pkgaws.ToPolicyDocument(instance, r.oidcProviderArn)
			if err != nil {
				return ctrl.Result{}, err
			}
			raw, err := json.Marshal(doc)
			if err != nil {
				return ctrl.Result{}, err
			}
			policy := string(raw)
			out, err := r.CreateRole(ctx, &iam.CreateRoleInput{
				RoleName:                 aws.String(name),
				AssumeRolePolicyDocument: aws.String(policy),
			})
			if err != nil {
				return ctrl.Result{}, err
			}
			r.notify.Created(aws.ToString(out.Role.RoleName))
			*upstream = *out.Role
		} else {
			return ctrl.Result{}, err
		}
	} else {
		*upstream = *out.Role
	}

	rawDoc, err := url.QueryUnescape(aws.ToString(upstream.AssumeRolePolicyDocument))
	if err != nil {
		return ctrl.Result{}, err
	}
	current := pkgaws.PolicyDocument{}
	if err := json.Unmarshal([]byte(rawDoc), &current); err != nil {
		// This would mean the upstream policy document isn't valid which isn't really feasible
		// so probably the PolicyDocument class is broken
		r.notify.InvalidPolicyDocument(aws.ToString(upstream.RoleName))
		return ctrl.Result{}, err
	}
	doc, err := pkgaws.ToPolicyDocument(instance, r.oidcProviderArn)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !reflect.DeepEqual(doc, current) {
		raw, err := json.Marshal(doc)
		if err != nil {
			return ctrl.Result{}, err
		}
		policy := string(raw)
		_, err = r.IamRoleService.UpdateAssumeRolePolicy(ctx, &iam.UpdateAssumeRolePolicyInput{
			RoleName:       upstream.RoleName,
			PolicyDocument: aws.String(policy),
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		r.notify.TrustPolicyUpdated(aws.ToString(upstream.RoleName))
	}

	instance.Status.RoleArn = aws.ToString(upstream.Arn)
	if err := k8.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IamRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IamRole{}).
		Complete(r)
}

func roleName(instance *v1alpha1.IamRole) string {
	name := instance.GetNamespace() + "-" + instance.GetName()
	return name
}
