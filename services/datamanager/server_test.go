package datamanager_test

import (
	"context"
	"errors"
	"testing"

	pb "go.viam.com/api/service/datamanager/v1"
	"go.viam.com/test"

	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/datamanager"
	"go.viam.com/rdk/subtype"
	"go.viam.com/rdk/testutils/inject"
)

func newServer(resourceMap map[resource.Name]interface{}) (pb.DataManagerServiceServer, error) {
	omSvc, err := subtype.New(resourceMap)
	if err != nil {
		return nil, err
	}
	return datamanager.NewServer(omSvc), nil
}

func TestServerSync(t *testing.T) {
	tests := map[string]struct {
		resourceMap   map[resource.Name]interface{}
		expectedError error
	}{
		"missing datamanager": {
			resourceMap:   map[resource.Name]interface{}{},
			expectedError: errors.New("resource \"rdk:service:data_manager/DataManager1\" not found"),
		},
		"not datamanager": {
			resourceMap: map[resource.Name]interface{}{
				datamanager.Named(testDataManagerServiceName): "not datamanager",
			},
			expectedError: datamanager.NewUnimplementedInterfaceError("string"),
		},
		"returns error": {
			resourceMap: map[resource.Name]interface{}{
				datamanager.Named(testDataManagerServiceName): &inject.DataManagerService{
					SyncFunc: func(
						ctx context.Context,
					) error {
						return errors.New("fake sync error")
					},
				},
			},
			expectedError: errors.New("fake sync error"),
		},
		"returns response": {
			resourceMap: map[resource.Name]interface{}{
				datamanager.Named(testDataManagerServiceName): &inject.DataManagerService{
					SyncFunc: func(
						ctx context.Context,
					) error {
						return nil
					},
				},
			},
			expectedError: nil,
		},
	}

	syncRequest := &pb.SyncRequest{Name: testDataManagerServiceName}
	// put resource
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resourceMap := tc.resourceMap
			server, err := newServer(resourceMap)
			test.That(t, err, test.ShouldBeNil)
			_, err = server.Sync(context.Background(), syncRequest)
			if tc.expectedError != nil {
				test.That(t, err, test.ShouldBeError, tc.expectedError)
			} else {
				test.That(t, err, test.ShouldBeNil)
			}
		})
	}
}
