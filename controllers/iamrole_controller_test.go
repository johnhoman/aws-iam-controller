package controllers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IamRoleController", func() {
	var mgr ctrl.Manager
	var ctx context.Context
	var cancel context.CancelFunc
	var ns *corev1.Namespace
	var k8sClient client.Client
	var c *EventuallyClient
	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		ns = &corev1.Namespace{}
		ns.SetName("testspace-" + uuid.New().String()[:8])

		k8sClient = client.NewNamespacedClient(k8s, ns.GetName())
		c = NewEventuallyClient(k8sClient)
		NewEventuallyClient(k8s).ExpectCreate(ctx, ns).Should(Succeed())

		var err error
		mgr, err = ctrl.NewManager(cfg, ctrl.Options{
			Scheme:                 scheme.Scheme,
			LeaderElection:         false,
			MetricsBindAddress:     "0",
			HealthProbeBindAddress: "0",
			Namespace:              ns.GetName(),
		})
		Expect(err).Should(Succeed())
		err = (&IamRoleReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}).SetupWithManager(mgr)
		Expect(err).Should(Succeed())
		go func() {
			defer GinkgoRecover()
			Expect(mgr.Start(ctx)).ToNot(HaveOccurred(), "failed to run manager")
		}()
	})
	AfterEach(func() { cancel() })
	It("Adds a finalizer", func() {
		instance := &v1alpha1.IamRole{}
		instance.SetName("adds-a-finalizer")
		instance.SetFinalizers([]string{"keep"})
		c.ExpectCreate(ctx, instance).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		c.ExpectGetWhen(ctx, types.NamespacedName{Name: "adds-a-finalizer"}, instance, func(obj client.Object) bool {
			if len(obj.GetFinalizers()) > 1 {
				return true
			}
			return false
		}).Should(Succeed())
		Expect(instance.Finalizers).Should(And(ContainElement(Finalizer), ContainElement("keep")))
		Expect(instance.ManagedFields[0].Manager).To(Equal("aws-iam-controller"))
		Expect(instance.ManagedFields[1].Manager).ToNot(Equal("aws-iam-controller"))

		managed := map[string]interface{}{
			"f:metadata": map[string]interface{}{
				"f:finalizers": map[string]interface{}{
					fmt.Sprintf(`v:"%s"`, Finalizer): map[string]interface{}{},
				},
			},
		}
		raw, _ := json.Marshal(managed)
		Expect(instance.ManagedFields[0].FieldsV1.Raw).To(Equal(raw))
	})
	It("Adds a finalizer forces ownership", func() {
		Skip("this path can't be reached in the controller currently")
		instance := &v1alpha1.IamRole{}
		instance.SetName("adds-a-finalizer")
		instance.SetFinalizers([]string{"keep", Finalizer})
		c.ExpectCreate(ctx, instance).Should(Succeed())
		Expect(instance.Finalizers).Should(And(ContainElement(Finalizer), ContainElement("keep")))
		Expect(instance.ManagedFields).To(HaveLen(1))
		Expect(instance.ManagedFields[0].Manager).ToNot(Equal("aws-iam-controller"))

		instance = &v1alpha1.IamRole{}
		c.ExpectGetWhen(ctx, types.NamespacedName{Name: "adds-a-finalizer"}, instance, func(obj client.Object) bool {
			return len(obj.GetManagedFields()) > 1
		}).Should(Succeed())
		Expect(instance.Finalizers).Should(And(ContainElement(Finalizer), ContainElement("keep")))
		Expect(instance.ManagedFields).To(HaveLen(2))
		Expect(instance.ManagedFields[0].Manager).To(Equal("aws-iam-controller"))
		Expect(instance.ManagedFields[1].Manager).ToNot(Equal("aws-iam-controller"))
	})
	It("Should remove a finalizer", func() {
		instance := &v1alpha1.IamRole{}
		instance.SetName("remove-a-finalizer")
		instance.SetFinalizers([]string{"keep"})
		c.ExpectCreate(ctx, instance).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		c.ExpectGetWhen(ctx, types.NamespacedName{Name: "remove-a-finalizer"}, instance, func(obj client.Object) bool {
			if len(obj.GetFinalizers()) > 1 {
				return true
			}
			return false
		}).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		instance.SetName("remove-a-finalizer")
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

		instance = &v1alpha1.IamRole{}
		c.ExpectGetWhen(ctx, types.NamespacedName{Name: "remove-a-finalizer"}, instance, func(obj client.Object) bool {
			if len(obj.GetFinalizers()) == 1 {
				return true
			}
			return false
		}).Should(Succeed())
		Expect(instance.Finalizers).Should(ContainElement("keep"))
		Expect(instance.Finalizers).ShouldNot(ContainElement(Finalizer))
		Expect(instance.ManagedFields[0].Manager).ToNot(Equal("aws-iam-controller"))
	})
})
