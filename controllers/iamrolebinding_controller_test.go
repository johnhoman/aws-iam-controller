package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"github.com/johnhoman/aws-iam-controller/pkg/bindmanager"
	"github.com/johnhoman/controller-tools/manager"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IamrolebindingController", func() {
	var mgr manager.IntegrationTest
	var iamService pkgaws.IamService
	var roleService iamrole.Interface
	var issuerUrl string
	var oidcArn string
	var doc map[string]interface{}
	BeforeEach(func() {
		issuerUrl = "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B716D3041E"
		oidcArn = fmt.Sprintf("arn:aws:iam::012345678912:oidc-provider/%s", issuerUrl)

		iamService = newIamService()
		roleService = iamrole.New(iamService, "controller-test")

		doc = defaultPolicy()
		raw, err := json.Marshal(doc)
		Expect(err).Should(Succeed())

		mgr = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		Expect((&IamRoleReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			notify:        &notifier{},
			roleService:   roleService,
			defaultPolicy: string(raw),
		}).SetupWithManager(mgr)).Should(Succeed())
		Expect((&IamRoleBindingReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			roleService:   roleService,
			oidcArn:       oidcArn,
			issuerUrl:     issuerUrl,
			EventRecorder: mgr.GetEventRecorderFor("controller.iamrolebinding.test"),
		}).SetupWithManager(mgr)).Should(Succeed())
		mgr.StartManager()
	})
	AfterEach(func() { mgr.StopManager() })
	When("The resource exists", func() {
		var instance *v1alpha1.IamRoleBinding
		var role *v1alpha1.IamRole
		var sa *corev1.ServiceAccount
		BeforeEach(func() {
			name := "iam-role-binding-test"

			role = &v1alpha1.IamRole{}
			role.SetName(name)

			sa = &corev1.ServiceAccount{}
			sa.SetName(name)

			instance = &v1alpha1.IamRoleBinding{}
			instance.SetName(name)
			instance.RoleRef = rbacv1.RoleRef{
				APIGroup: "aws.jackhoman.com",
				Kind:     role.Kind,
				Name:     name,
			}
			instance.Subjects = []rbacv1.Subject{{
				APIGroup:  sa.GroupVersionKind().Group,
				Kind:      sa.Kind,
				Name:      sa.GetName(),
				Namespace: sa.GetNamespace(),
			}}
			mgr.Eventually().Create(instance).Should(Succeed())
		})
		When("the role exists", func() {
			BeforeEach(func() {
				mgr.Eventually().Create(role).Should(Succeed())
				mgr.Eventually().GetWhen(types.NamespacedName{Name: role.GetName()}, role, func(obj client.Object) bool {
					return len(obj.(*v1alpha1.IamRole).Status.RoleArn) > 0
				}).Should(Succeed())
			})
			When("the subject exists", func() {
				BeforeEach(func() {
					mgr.Eventually().Create(sa).Should(Succeed())
				})
				When("the subject is a service account", func() {
					When("the role trusts the service account", func() {
						BeforeEach(func() {
							doc = defaultPolicy()
							statements := doc["Statement"].([]interface{})
							statements = append(statements, map[string]interface{}{
								"Sid": fmt.Sprintf(
									"AllowServiceAccount-%s-%s",
									sa.GetNamespace(),
									sa.GetName(),
								),
								"Effect":    "Allow",
								"Principal": map[string]interface{}{"Federated": oidcArn},
								"Action":    "sts:AssumeRoleWithWebIdentity",
								"Condition": map[string]interface{}{
									"StringEquals": map[string]interface{}{
										issuerUrl: fmt.Sprintf(
											"system:serviceaccount:%s:%s",
											sa.GetNamespace(),
											sa.GetName(),
										),
									},
								},
							})
							doc["Statement"] = statements
							raw, err := json.Marshal(doc)
							Expect(err).ShouldNot(HaveOccurred())
							_, err = roleService.Update(mgr.GetContext(), &iamrole.UpdateOptions{
								Name:           role.GetName(),
								PolicyDocument: string(raw),
							})
							Expect(err).Should(Succeed())
						})
						It("Should annotate the service account", func() {
							obj := &corev1.ServiceAccount{}
							mgr.Eventually().GetWhen(types.NamespacedName{Name: sa.GetName()}, obj, func(obj client.Object) bool {
								return len(obj.GetAnnotations()) > 0
							}).ShouldNot(HaveOccurred())
							Expect(obj.GetAnnotations()).Should(HaveKeyWithValue("eks.amazonaws.com/role-arn", role.Status.RoleArn))
						})
					})
					When("the role doesn't trust the service account", func() {
						It("Should update the trust policy", func() {
							Eventually(func() bool {
								bm := bindmanager.New(roleService, oidcArn)
								binding := bindmanager.Binding{ServiceAccount: sa, Role: role}
								ok, err := bm.IsBound(mgr.GetContext(), &binding)
								if err != nil {
									return false
								}
								return ok
							}).Should(BeTrue())
							r, err := roleService.Get(mgr.GetContext(), &iamrole.GetOptions{Name: role.GetName()})
							Expect(err).ShouldNot(HaveOccurred())
							Expect(r.TrustPolicy).Should(And(
								ContainSubstring("system:serviceaccount"),
								ContainSubstring(sa.GetName()),
								ContainSubstring(sa.GetNamespace()),
								ContainSubstring(oidcArn),
							))
						})
					})
					When("the service account is annotated", func() {
						When("the service account is annotated by another role binding", func() {
							It("Should not annotate the service account", func() {
								Skip("not implemented")
							})
							It("Should not trust the service account", func() {
								Skip("not implemented")
							})
						})
						When("the service account is annotated by the current role binding", func() {
							It("Should have a managed field entry for the service account", func() {
								Skip("not implemented")
							})
						})
					})
					When("the service account is not annotated", func() {
						It("Should annotate the service account", func() {
							Skip("not implemented")
						})
					})
				})
			})
			When("the subject does not exist", func() {
				When("the role binding trusts the subject", func() {
					It("Should update the trust policy", func() {
						Skip("not implemented")
					})
				})
				When("the role binding does not trust the subject", func() {
					It("Do not update the trust policy", func() {
						Skip("not implemented")
					})
				})
			})
		})
		When("The role does not exist", func() {
			When("the subject is a service account", func() {
				When("the service account is not annotated", func() {
					It("does not annotate the service account", func() {
						Skip("not implemented")
					})
				})
			})
		})
	})
})
