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
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8s client.Client
var testEnv *envtest.Environment

type EventuallyClient struct {
	Client client.Client
}

func (a *EventuallyClient) ExpectCreate(ctx context.Context, obj client.Object) AsyncAssertion {
	return Eventually(func() error {
		key := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
		err := a.Client.Get(ctx, key, obj)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		if apierrors.IsNotFound(err) {
			err = a.Client.Create(ctx, obj)
			if err != nil {
				return err
			}
		}
		return err
	})
}

func (a *EventuallyClient) ExpectGet(ctx context.Context, key types.NamespacedName, obj client.Object) AsyncAssertion {
	return Eventually(func() error {
		return a.Client.Get(ctx, key, obj)
	})
}

func (a *EventuallyClient) ExpectGetWhen(ctx context.Context, key types.NamespacedName, obj client.Object, predicate func(obj client.Object) bool) AsyncAssertion {
	return Eventually(func() error {
		err := a.Client.Get(ctx, key, obj)
		if err != nil {
			return err
		}
		if !predicate(obj) {
			return errors.New(fmt.Sprintf("predicate failed: %#v", obj))
		}
		return nil
	})
}

func (a *EventuallyClient) ExpectUpdate(ctx context.Context, obj client.Object) AsyncAssertion {
	return Eventually(func() error {
		version := obj.GetResourceVersion()
		err := a.Client.Update(ctx, obj)
		if err != nil {
			return err
		}
		key := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
		err = a.Client.Get(ctx, key, obj)
		if err != nil {
			return err
		}
		if obj.GetResourceVersion() != version {
			return errors.New("waiting for update to propagate")
		}
		return nil
	})
}

func NewEventuallyClient(client client.Client) *EventuallyClient {
	return &EventuallyClient{Client: client}
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8s, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
