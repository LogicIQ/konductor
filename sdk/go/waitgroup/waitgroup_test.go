package waitgroup

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

func TestAdd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
		Status: syncv1.WaitGroupStatus{
			Counter: 0,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg).
		WithStatusSubresource(&syncv1.WaitGroup{}).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")
	ctx := context.Background()

	err := Add(client, ctx, "test-wg", 3)
	require.NoError(t, err)

	counter, err := GetCounter(client, ctx, "test-wg")
	require.NoError(t, err)
	assert.Equal(t, int32(3), counter)
}

func TestDone(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
		Status: syncv1.WaitGroupStatus{
			Counter: 3,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg).
		WithStatusSubresource(&syncv1.WaitGroup{}).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")
	ctx := context.Background()

	err := Done(client, ctx, "test-wg")
	require.NoError(t, err)

	counter, err := GetCounter(client, ctx, "test-wg")
	require.NoError(t, err)
	assert.Equal(t, int32(2), counter)
}

func TestCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")
	ctx := context.Background()

	err := Create(client, ctx, "test-wg", konductor.WithTTL(5*time.Minute))
	require.NoError(t, err)
}

func TestList(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg1 := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wg1",
			Namespace: "default",
		},
	}
	wg2 := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wg2",
			Namespace: "default",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg1, wg2).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")
	ctx := context.Background()

	wgs, err := List(client, ctx)
	require.NoError(t, err)
	assert.Len(t, wgs, 2)
}
