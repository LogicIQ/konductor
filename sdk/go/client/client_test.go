package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestNewFromClient(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	tests := []struct {
		name              string
		namespace         string
		expectedNamespace string
	}{
		{
			name:              "with namespace",
			namespace:         "test-ns",
			expectedNamespace: "test-ns",
		},
		{
			name:              "empty namespace defaults to default",
			namespace:         "",
			expectedNamespace: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewFromClient(k8sClient, tt.namespace)
			assert.Equal(t, tt.expectedNamespace, client.Namespace())
			assert.Equal(t, k8sClient, client.K8sClient())
		})
	}
}

func TestClient_WithNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	client := NewFromClient(k8sClient, "original")

	newClient := client.WithNamespace("new-ns")

	assert.Equal(t, "original", client.Namespace())
	assert.Equal(t, "new-ns", newClient.Namespace())
	assert.Equal(t, k8sClient, newClient.K8sClient())
}

func TestOptions(t *testing.T) {
	opts := &Options{}

	WithTTL(300)(opts)
	WithTimeout(60)(opts)
	WithPriority(5)(opts)
	WithHolder("test-holder")(opts)

	assert.Equal(t, int64(300), opts.TTL.Nanoseconds())
	assert.Equal(t, int64(60), opts.Timeout.Nanoseconds())
	assert.Equal(t, int32(5), opts.Priority)
	assert.Equal(t, "test-holder", opts.Holder)
}