package controllers

import (
	"context"
	"fmt"
	awsv1alpha1 "github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type IamRoleBindingReconciler struct {
	client.Client
	*runtime.Scheme
	record.EventRecorder
}

const (
	ServiceAccountFinalizer  = "aws.jackhoman.com/free-service-account"
	ServiceAccountAnnotation = "eks.amazonaws.com/role-arn"
)

//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamrolebindings/finalizers,verbs=update
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles,verbs=get;list;watch;
//+kubebuilder:rbac:groups=aws.jackhoman.com,resources=iamroles/status,verbs=get;
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;patch;update;

func (r *IamRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	k8s := client.NewNamespacedClient(r.Client, req.Namespace)

	instance := &awsv1alpha1.IamRoleBinding{}
	if err := k8s.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, err
	}
	if !instance.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(instance, ServiceAccountFinalizer) {
			if err := r.finalize(ctx, instance); err != nil {
				logger.Error(err, "unable to finalize resource")
				return ctrl.Result{}, err
			}
			patch := client.MergeFrom(instance.DeepCopy())
			controllerutil.RemoveFinalizer(instance, ServiceAccountFinalizer)
			if err := k8s.Patch(ctx, instance, patch); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	if !controllerutil.ContainsFinalizer(instance, ServiceAccountFinalizer) {
		patch := &unstructured.Unstructured{Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"finalizers": []string{ServiceAccountFinalizer},
			},
		}}
		patch.SetGroupVersionKind(instance.GroupVersionKind())
		patch.SetName(instance.GetName())
		if err := k8s.Patch(ctx, patch, client.Apply, client.FieldOwner("controller.iamrolebinding")); err != nil {
			logger.Error(err, "unable add finalizer")
			return ctrl.Result{}, err
		}
	}
	if len(instance.Status.BoundServiceAccountRef.Name) > 0 && instance.Status.BoundServiceAccountRef != instance.Spec.ServiceAccountRef {
		logger.Info("sync service accounts",
			"have",
			instance.Status.BoundServiceAccountRef,
			"want",
			instance.Spec.ServiceAccountRef,
		)
		if err := r.finalize(ctx, instance); err != nil {
			logger.Error(err, "unable to remove binding from service account")
			return ctrl.Result{}, err
		}
	}

	iamRole := &awsv1alpha1.IamRole{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name: instance.Spec.IamRoleRef.Name,
		// TODO: remove namespace when iam role is switched to cluster scope
		Namespace: instance.Namespace,
	}, iamRole); err != nil {
		logger.Error(err, "unable to get IamRole")
		return ctrl.Result{}, err
	}

	if len(iamRole.Status.RoleArn) == 0 {
		err := fmt.Errorf("")
		logger.Error(err, "IamRole is missing role-arn from status")
		return ctrl.Result{}, err
	}

	if err := r.bindServiceAccount(ctx, instance, iamRole); err != nil {
		logger.Error(err, "unable to bind service account")
		return ctrl.Result{}, err
	}
	if instance.Status.BoundServiceAccountRef != instance.Spec.ServiceAccountRef || instance.Status.BoundIamRoleArn != iamRole.Status.RoleArn {
		patch := client.MergeFrom(instance.DeepCopy())
		instance.Status.BoundServiceAccountRef = instance.Spec.ServiceAccountRef
		instance.Status.BoundIamRoleArn = iamRole.Status.RoleArn
		if err := k8s.Status().Patch(ctx, instance, patch); err != nil {
			logger.Error(err, "unable to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *IamRoleBindingReconciler) bindServiceAccount(
	ctx context.Context,
	instance *awsv1alpha1.IamRoleBinding,
	iamRole *awsv1alpha1.IamRole,
) error {
	k8s := client.NewNamespacedClient(r.Client, instance.GetNamespace())
	keysAndValues := []interface{}{
		"serviceAccountName",
		instance.Spec.ServiceAccountRef,
		"iamRoleName",
		instance.Spec.IamRoleRef,
	}
	logger := log.FromContext(ctx).WithValues(keysAndValues...)

	serviceAccount := &corev1.ServiceAccount{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: instance.Spec.ServiceAccountRef.Name}, serviceAccount); err != nil {
		logger.Error(err, "unable to get service account")
		return err
	}

	annotations := serviceAccount.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	existing, ok := annotations[ServiceAccountAnnotation]
	if ok && existing != iamRole.Status.RoleArn {
		// Something else is bound to this instance
		r.Event(instance, corev1.EventTypeWarning, "Conflict", "Service account is already bound to an iam role")
		return fmt.Errorf("unable to bind service account because of conflict")
	}
	if !ok {
		patch := client.MergeFrom(serviceAccount.DeepCopy())
		annotations[ServiceAccountAnnotation] = iamRole.Status.RoleArn
		serviceAccount.SetAnnotations(annotations)
		if err := k8s.Patch(ctx, serviceAccount, patch); err != nil {
			logger.Error(err, "unable to add service account annotation")
			return err
		}
	}
	return nil
}

func (r *IamRoleBindingReconciler) finalize(ctx context.Context, instance *awsv1alpha1.IamRoleBinding) error {
	// Remove bound service account annotation
	k8s := client.NewNamespacedClient(r.Client, instance.GetNamespace())
	logger := log.FromContext(ctx, "finalize", instance.Status.BoundServiceAccountRef)
	if instance.Status.BoundServiceAccountRef.Name == "" {
		return nil
	}
	serviceAccount := &corev1.ServiceAccount{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: instance.Status.BoundServiceAccountRef.Name}, serviceAccount); err != nil {
		logger.Error(err, "unable to get service account")
		return client.IgnoreNotFound(err)
	}
	if serviceAccount.GetAnnotations() != nil {
		if arn, ok := serviceAccount.GetAnnotations()[ServiceAccountAnnotation]; ok {
			if arn == instance.Status.BoundIamRoleArn {
				// remove the annotation
				patch := client.MergeFrom(serviceAccount.DeepCopy())
				annotations := serviceAccount.GetAnnotations()
				delete(annotations, ServiceAccountAnnotation)
				serviceAccount.SetAnnotations(annotations)
				if err := k8s.Patch(ctx, serviceAccount, patch); err != nil {
					logger.Error(err, "unable to remove annotation from service account")
					return err
				}
			}
		}
	}
	return nil
}

func (r *IamRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.IamRoleBinding{}).
		Complete(r)
}

var _ reconcile.Reconciler = &IamRoleBindingReconciler{}
