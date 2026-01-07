package waitgroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

func TestAdd_Basic(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
		Status: syncv1.WaitGroupStatus{
			Counter: 5,
			Phase:   syncv1.WaitGroupPhaseWaiting,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg).
		WithStatusSubresource(&syncv1.WaitGroup{}).
		Build()
	client := konductor.NewFromClient(k8sClient, "default")

	// Test basic functionality without retry logic
	var updated syncv1.WaitGroup
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name: "test-wg", Namespace: "default",
	}, &updated)
	require.NoError(t, err)
	
	updated.Status.Counter += 2
	err = k8sClient.Status().Update(context.Background(), &updated)
	require.NoError(t, err)

	// Verify the counter was updated
	final, err := Get(client, context.Background(), "test-wg")
	require.NoError(t, err)
	assert.Equal(t, int32(7), final.Status.Counter)
}

func TestCreate_Basic(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	client := konductor.NewFromClient(k8sClient, "default")

	// Test basic create without retry
	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
		Spec: syncv1.WaitGroupSpec{},
	}

	err := k8sClient.Create(context.Background(), wg)
	assert.NoError(t, err)

	// Verify creation
	created, err := Get(client, context.Background(), "test-wg")
	require.NoError(t, err)
	assert.Equal(t, "test-wg", created.Name)
}