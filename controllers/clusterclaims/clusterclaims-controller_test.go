package clusterlcaims

import (
	"context"
	"testing"
	"time"

	mcv1 "github.com/open-cluster-management/api/cluster/v1"
	kacv1 "github.com/open-cluster-management/klusterlet-addon-controller/pkg/apis/agent/v1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const CC_NAME = "my-clusterclaim"
const CC_NAMESPACE = "my-pool"
const CP_NAME = "chlorine-and-salt"
const CLUSTER01 = "cluster01"
const NO_CLUSTER = ""

var s = scheme.Scheme

func init() {
	corev1.SchemeBuilder.AddToScheme(s)
	hivev1.SchemeBuilder.AddToScheme(s)
	mcv1.AddToScheme(s)
	kacv1.SchemeBuilder.AddToScheme(s)
}

func getRequest() ctrl.Request {
	return getRequestWithNamespaceName(CC_NAMESPACE, CC_NAME)
}

func getRequestWithNamespaceName(rNamespace string, rName string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: getNamespaceName(rNamespace, rName),
	}
}

func getNamespaceName(namespace string, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

func GetClusterClaim(namespace string, name string, clusterName string) *hivev1.ClusterClaim {
	return &hivev1.ClusterClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"usage": "production",
			},
		},
		Spec: hivev1.ClusterClaimSpec{
			ClusterPoolName: "make-believe",
			Namespace:       clusterName,
		},
	}
}

func GetClusterClaimsReconciler() *ClusterClaimsReconciler {

	// Log levels: DebugLevel  DebugLevel
	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.Level(zapcore.DebugLevel)))

	return &ClusterClaimsReconciler{
		Client: clientfake.NewFakeClientWithScheme(s),
		Log:    ctrl.Log.WithName("controllers").WithName("ClusterClaimsReconciler"),
		Scheme: s,
	}
}

func TestReconcileClusterClaims(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")

	assert.Equal(t, mc.Labels["name"], CC_NAME, "label name should equal clusterClaim name")
	assert.Equal(t, mc.Labels["vendor"], "OpenShift", "label vendor should equal OpenShift")
	assert.Equal(t, mc.Labels["usage"], "production", "label usage should equal production")

	var kac kacv1.KlusterletAddonConfig
	err = ccr.Client.Get(ctx, getNamespaceName(CLUSTER01, CLUSTER01), &kac)
	assert.Nil(t, err, "nil, when klusterletAddonConfig resource is retrieved")

}

func TestReconcileClusterClaimsLabelCopy(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	var mc mcv1.ManagedCluster
	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), &mc)
	assert.Nil(t, err, "nil, when managedCluster resource is retrieved")

	assert.Equal(t, mc.Labels["name"], CC_NAME, "label name should equal clusterClaim name")
	assert.Equal(t, mc.Labels["vendor"], "OpenShift", "label vendor should equal OpenShift")
	assert.Equal(t, mc.Labels["usage"], "production", "label usage should equal production")

	var kac kacv1.KlusterletAddonConfig
	err = ccr.Client.Get(ctx, getNamespaceName(CLUSTER01, CLUSTER01), &kac)
	assert.Nil(t, err, "nil, when klusterletAddonConfig resource is retrieved")

	assert.Equal(t, kac.Spec.ClusterLabels["vendor"], "OpenShift", "Check clusterLabels set")
}
func TestReconcileExistingManagedCluster(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	ccr.Client.Create(ctx, GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01), &client.CreateOptions{})

	mc := &mcv1.ManagedCluster{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01}}
	kac := &kacv1.KlusterletAddonConfig{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01, Namespace: CLUSTER01}}

	ccr.Client.Create(ctx, mc, &client.CreateOptions{})
	ccr.Client.Create(ctx, kac, &client.CreateOptions{})

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

}

func TestReconcileDeletedClusterClaim(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	cc := GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01)

	cc.DeletionTimestamp = &v1.Time{time.Now()}

	ccr.Client.Create(ctx, cc, &client.CreateOptions{})

	mc := &mcv1.ManagedCluster{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01}}
	kac := &kacv1.KlusterletAddonConfig{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01, Namespace: CLUSTER01}}

	ccr.Client.Create(ctx, mc, &client.CreateOptions{})
	ccr.Client.Create(ctx, kac, &client.CreateOptions{})

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), mc)
	assert.NotNil(t, err, "nil, when managedCluster resource is retrieved")
	assert.Contains(t, err.Error(), " not found", "error should be NotFound")

	err = ccr.Client.Get(ctx, getNamespaceName(CLUSTER01, CLUSTER01), kac)
	assert.NotNil(t, err, "nil, when klusterletAddonConfig resource is retrieved")
	assert.Contains(t, err.Error(), " not found", "error should be NotFound")

}

func TestReconcileDeletedClusterClaimWithAlreadyDeletingManagedCluster(t *testing.T) {

	ctx := context.Background()

	ccr := GetClusterClaimsReconciler()

	cc := GetClusterClaim(CC_NAMESPACE, CC_NAME, CLUSTER01)

	cc.DeletionTimestamp = &v1.Time{time.Now()}

	ccr.Client.Create(ctx, cc, &client.CreateOptions{})

	mc := &mcv1.ManagedCluster{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01}}
	kac := &kacv1.KlusterletAddonConfig{ObjectMeta: v1.ObjectMeta{Name: CLUSTER01, Namespace: CLUSTER01}}

	mc.DeletionTimestamp = &v1.Time{time.Now()}
	kac.DeletionTimestamp = &v1.Time{time.Now()}
	ccr.Client.Create(ctx, mc, &client.CreateOptions{})
	ccr.Client.Create(ctx, kac, &client.CreateOptions{})

	_, err := ccr.Reconcile(getRequest())

	assert.Nil(t, err, "nil, when clusterClaim is found reconcile was successful")

	err = ccr.Client.Get(ctx, getNamespaceName("", CLUSTER01), mc)
	assert.Nil(t, err, "nil, when managedCluster resource is skipped because it is already deleting")

	err = ccr.Client.Get(ctx, getNamespaceName(CLUSTER01, CLUSTER01), kac)
	assert.Nil(t, err, "nil, when klusterletAddonConfig resource is skipped because it is already deleting")

}
