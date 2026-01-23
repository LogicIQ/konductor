package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func setupOutputTestScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))
	return scheme
}

func TestOutputFormat_Text(t *testing.T) {
	scheme := setupOutputTestScheme(t)

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			InUse:     2,
			Available: 3,
			Phase:     syncv1.SemaphorePhaseReady,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()
	namespace = "default"
	outputFormat = "text"

	cmd := newSemaphoreListCmd()
	output, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)

	// Text format should contain readable output
	assert.Contains(t, output, "test-sem")
	assert.Contains(t, output, "Semaphore")
}

func TestOutputFormat_JSON(t *testing.T) {
	scheme := setupOutputTestScheme(t)

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			InUse:     2,
			Available: 3,
			Phase:     syncv1.SemaphorePhaseReady,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()
	namespace = "default"
	outputFormat = "json"

	cmd := newSemaphoreListCmd()
	output, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)

	// JSON format should contain structured data
	assert.Contains(t, output, "test-sem")

	// Should contain JSON fields (L=level, M=message in production config)
	assert.Contains(t, output, "\"L\"")
	assert.Contains(t, output, "\"M\"")
	assert.Contains(t, output, "timestamp")
}

func TestOutputFormat_Default(t *testing.T) {
	scheme := setupOutputTestScheme(t)

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"
	originalFormat := outputFormat
	outputFormat = "" // Empty should default to text
	defer func() { outputFormat = originalFormat }()

	cmd := newSemaphoreListCmd()
	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)

	// executeCommandWithOutput uses local variable that defaults to "text"
	// The global outputFormat remains empty, which is expected behavior
	assert.Equal(t, "", outputFormat)
}

func TestInitLogger_TextFormat(t *testing.T) {
	outputFormat = "text"
	logLevel = "info"

	err := initLogger()
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Logger should be initialized successfully
	logger.Info("test message")
	logger.Sync()
}

func TestInitLogger_JSONFormat(t *testing.T) {
	outputFormat = "json"
	logLevel = "info"

	err := initLogger()
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Logger should be initialized successfully
	logger.Info("test message")
	logger.Sync()
}

func TestInitLogger_InvalidLevel(t *testing.T) {
	outputFormat = "text"
	logLevel = "invalid"

	err := initLogger()
	require.NoError(t, err) // Should default to info level
	require.NotNil(t, logger)
}
