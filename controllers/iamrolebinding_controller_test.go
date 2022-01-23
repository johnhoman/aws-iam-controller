package controllers

import (
	pkgaws "github.com/johnhoman/aws-iam-controller/pkg/aws"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"github.com/johnhoman/controller-tools/manager"
	"k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IamrolebindingController", func() {
	var mgr manager.IntegrationTest
	var iamService pkgaws.IamService
	BeforeEach(func() {
		iamService = newIamService()

		mgr = manager.IntegrationTestBuilder().
			WithScheme(scheme.Scheme).
			Complete(cfg)

		Expect((&IamRoleReconciler{
			Client:      mgr.GetClient(),
			Scheme:      mgr.GetScheme(),
			notify:      &notifier{},
			roleService: iamrole.New(iamService, "controller-test"),
		}).SetupWithManager(mgr)).Should(Succeed())
		Expect((&IamRoleBindingReconciler{
			Client:      mgr.GetClient(),
			Scheme:      mgr.GetScheme(),
			roleService: iamrole.New(iamService, "controller-test"),
		}).SetupWithManager(mgr)).Should(Succeed())
		mgr.StartManager()
	})
	AfterEach(func() { mgr.StopManager() })
})
