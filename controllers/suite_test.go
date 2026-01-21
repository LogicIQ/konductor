package controllers

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

type ControllerTestSuite struct {
	suite.Suite
	scheme *runtime.Scheme
}

func (suite *ControllerTestSuite) SetupSuite() {
	suite.scheme = runtime.NewScheme()
	suite.Require().NoError(syncv1.AddToScheme(suite.scheme))
}

func (suite *ControllerTestSuite) NewFakeClientBuilder(objs ...runtime.Object) *fake.ClientBuilder {
	return fake.NewClientBuilder().
		WithScheme(suite.scheme).
		WithRuntimeObjects(objs...).
		WithStatusSubresource(
			&syncv1.Semaphore{},
			&syncv1.Barrier{},
			&syncv1.Lease{},
			&syncv1.Gate{},
			&syncv1.Permit{},
			&syncv1.Arrival{},
			&syncv1.LeaseRequest{},
		)
}

func TestControllerSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}
