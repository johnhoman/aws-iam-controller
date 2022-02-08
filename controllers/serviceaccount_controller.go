package controllers

import (
	"context"
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	RoleArnAnnotation = "eks.amazonaws.com/role-arn"
)

type ServiceAccountReconciler struct {
	client.Client
	*runtime.Scheme
	record.EventRecorder
}

//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings,verbs=get;list;watch;
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=serviceaccounts/finalizers,verbs=update

func (r *ServiceAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	k8s := client.NewNamespacedClient(r.Client, req.Namespace)

	instance := corev1.ServiceAccount{}
	if err := k8s.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	bindings := v1alpha1.IamRoleBindingList{}
	if err := k8s.List(ctx, &bindings, client.MatchingFields{"spec.ServiceAccountRef": instance.GetName()}); err != nil {
		return ctrl.Result{}, err
	}
	if len(bindings.Items) == 0 {
		// if nothing references the service account then remove all annotations
		if instance.GetAnnotations() != nil {
			_, ok := instance.GetAnnotations()[RoleArnAnnotation]
			if ok {
				// Remove the annotation
				patch := client.MergeFrom(instance.DeepCopy())
				annotations := instance.GetAnnotations()
				delete(annotations, RoleArnAnnotation)
				instance.SetAnnotations(annotations)
				if err := k8s.Patch(ctx, &instance, patch); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
		return ctrl.Result{}, nil
	}
	if len(bindings.Items) > 1 {
		// if multiple bindings reference this service account notify something
		logger.Info("too many bindings reference this service account")
		if instance.GetAnnotations() != nil {
			_, ok := instance.GetAnnotations()[RoleArnAnnotation]
			if ok {
				// if one is already bound then use that one
			}
		}
		// if none are bound then use the service account binding created first
		bindings.Items = []v1alpha1.IamRoleBinding{bindings.Items[0]}
	}

	iamRole := v1alpha1.IamRole{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: bindings.Items[0].Spec.IamRoleRef}, &iamRole); err != nil {
		return ctrl.Result{}, err
	}

	arn := iamRole.Status.RoleArn
	if len(arn) == 0 {
		logger.Info("roleArn no provided", "role", iamRole.GetName())
		return ctrl.Result{Requeue: true}, nil
	}
	if instance.GetAnnotations() == nil {
		instance.SetAnnotations(map[string]string{})
	}
	ann, ok := instance.GetAnnotations()[RoleArnAnnotation]
	if !ok || ann != arn {
		patch := client.MergeFrom(instance.DeepCopy())
		annotations := instance.GetAnnotations()
		annotations[RoleArnAnnotation] = arn
		instance.SetAnnotations(annotations)
		if err := k8s.Patch(ctx, &instance, patch); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.IamRoleBinding{}, "spec.ServiceAccountRef", func(obj client.Object) []string {
		binding, ok := obj.(*v1alpha1.IamRoleBinding)
		if !ok {
			return []string{}
		}
		return []string{binding.Spec.ServiceAccountRef}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&corev1.ServiceAccount{},
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(_ event.CreateEvent) bool {
					return false
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					if _, ok := e.ObjectOld.GetAnnotations()[RoleArnAnnotation]; ok {
						return true
					}
					return false
				},
				GenericFunc: func(_ event.GenericEvent) bool {
					return false
				},
			}),
		).
		Watches(
			&source.Kind{Type: &v1alpha1.IamRoleBinding{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []ctrl.Request {
				binding, ok := obj.(*v1alpha1.IamRoleBinding)
				if ok {
					return []ctrl.Request{{
						NamespacedName: types.NamespacedName{
							Name:      binding.Spec.ServiceAccountRef,
							Namespace: binding.GetNamespace(),
						},
					}}
				}
				return []ctrl.Request{}
			}),
		).
		Complete(r)
}

var _ reconcile.Reconciler = &ServiceAccountReconciler{}
