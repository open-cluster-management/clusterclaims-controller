// Copyright Contributors to the Open Cluster Management project.

package main

import (
	"flag"
	"os"

	controller "github.com/jnpacker/clusterclaims-controller/controller/clusterpools"
	mcv1 "github.com/open-cluster-management/api/cluster/v1"
	kacv1 "github.com/open-cluster-management/klusterlet-addon-controller/pkg/apis/agent/v1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = hivev1.AddToScheme(scheme)
	_ = kacv1.SchemeBuilder.AddToScheme(scheme)
	_ = mcv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8383", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	// To run in debug change zapcore.InfoLevel to zapcore.DebugLevel
	ctrl.SetLogger(zap.New(zap.Level(zapcore.InfoLevel)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "clusterpools-controller.open-cluster-management.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.ClusterPoolsReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controller").WithName("ClusterPoolsReconciler"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
