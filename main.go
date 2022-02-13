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

package main

import (
	"context"
	"flag"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
	"github.com/johnhoman/aws-iam-controller/pkg/bindmanager"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	awsv1alpha1 "github.com/johnhoman/aws-iam-controller/api/v1alpha1"
	"github.com/johnhoman/aws-iam-controller/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme     = runtime.NewScheme()
	setupLog   = ctrl.Log.WithName("setup")
	denyPolicy = map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []interface{}{
			map[string]interface{}{
				"Sid":       "DenyAllAWS",
				"Effect":    "Deny",
				"Principal": map[string]interface{}{"AWS": "*"},
				"Action":    "sts:AssumeRole",
			},
		},
	}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(awsv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var path string
	var oidcArn string
	var awsRegion string
	var awsProfile string
	var enableWebhook bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&path, "resource-default-path", "", "The path prefix to use for creating IAM resources")
	flag.StringVar(&oidcArn, "oidc-arn", "", "The EKS cluster oidc provider")
	flag.StringVar(&awsRegion, "aws-region", "", "aws region")
	flag.StringVar(&awsProfile, "aws-profile", "", "aws shared credentials profile")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableWebhook, "enable-webhook", true, "Enable the webhook server")
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "4b7e85e7.jackhoman.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	options := make([]func(*config.LoadOptions) error, 0)
	if len(awsRegion) != 0 {
		options = append(options, config.WithRegion(awsRegion))
	}
	if len(awsProfile) > 0 {
		options = append(options, config.WithSharedConfigProfile(awsProfile))
	}
	cfg, err := config.LoadDefaultConfig(ctx, options...)
	if err != nil {
		setupLog.Error(err, "unable to load aws credentials")
		os.Exit(1)
	}

	if len(oidcArn) == 0 {
		setupLog.Info("missing required argument -oidc-arn")
		os.Exit(1)
	}

	client := iam.NewFromConfig(cfg)
	service := iamrole.New(client, path)

	raw, err := json.Marshal(denyPolicy)
	if err != nil {
		setupLog.Error(err, "unable to marshal provided default policy")
		os.Exit(1)
	}

	if err = (&controllers.IamRoleReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("controller.iamrole"),
		RoleService:   service,
		DefaultPolicy: string(raw),
		Manager:       bindmanager.New(service, oidcArn),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IamRole")
		os.Exit(1)
	}

	if enableWebhook {
		if err = (&v1alpha1.IamRole{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "IamRole")
			os.Exit(1)
		}
		if err = (&awsv1alpha1.IamRoleBinding{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "IamRoleBinding")
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
