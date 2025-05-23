package configuration

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	operatorclientsetfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	operatorexternalversions "github.com/openshift/client-go/operator/informers/externalversions"
)

func TestClusterVersionOperatorConfiguration_sync(t *testing.T) {
	tests := []struct {
		name                   string
		config                 operatorv1alpha1.ClusterVersionOperator
		expectedConfig         operatorv1alpha1.ClusterVersionOperator
		internalConfig         ClusterVersionOperatorConfiguration
		expectedInternalConfig ClusterVersionOperatorConfiguration
	}{
		{
			name: "first sync run correctly updates the status",
			config: operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.Normal,
				},
			},
			expectedConfig: operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.Normal,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 1,
				},
			},
			internalConfig: ClusterVersionOperatorConfiguration{
				desiredLogLevel:        operatorv1.Normal,
				lastObservedGeneration: 0,
			},
			expectedInternalConfig: ClusterVersionOperatorConfiguration{
				desiredLogLevel:        operatorv1.Normal,
				lastObservedGeneration: 1,
			},
		},
		{
			name: "sync updates observed generation correctly",
			config: operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 3,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.Normal,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 2,
				},
			},
			expectedConfig: operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 3,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.Normal,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 3,
				},
			},
			internalConfig: ClusterVersionOperatorConfiguration{
				desiredLogLevel:        operatorv1.Normal,
				lastObservedGeneration: 2,
			},
			expectedInternalConfig: ClusterVersionOperatorConfiguration{
				desiredLogLevel:        operatorv1.Normal,
				lastObservedGeneration: 3,
			},
		},
		{
			name: "sync updates desired log level correctly",
			config: operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 4,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.Trace,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 3,
				},
			},
			expectedConfig: operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 4,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.Trace,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 4,
				},
			},
			internalConfig: ClusterVersionOperatorConfiguration{
				desiredLogLevel:        operatorv1.Normal,
				lastObservedGeneration: 3,
			},
			expectedInternalConfig: ClusterVersionOperatorConfiguration{
				desiredLogLevel:        operatorv1.Trace,
				lastObservedGeneration: 4,
			},
		},
		{
			name: "number of not observed generations does not impact sync",
			config: operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 40,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.TraceAll,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 3,
				},
			},
			expectedConfig: operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 40,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.TraceAll,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 40,
				},
			},
			internalConfig: ClusterVersionOperatorConfiguration{
				desiredLogLevel:        operatorv1.Normal,
				lastObservedGeneration: 3,
			},
			expectedInternalConfig: ClusterVersionOperatorConfiguration{
				desiredLogLevel:        operatorv1.TraceAll,
				lastObservedGeneration: 40,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize testing logic
			client := operatorclientsetfake.NewClientset(&tt.config)
			tt.internalConfig.client = client.OperatorV1alpha1().ClusterVersionOperators()
			ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))

			// Run tested functionality
			if err := tt.internalConfig.sync(ctx, &tt.config); err != nil {
				t.Errorf("unexpected error %v", err)
			}

			// Verify results
			if tt.internalConfig.lastObservedGeneration != tt.expectedInternalConfig.lastObservedGeneration {
				t.Errorf("unexpected 'lastObservedGeneration' value; wanted=%v, got=%v", tt.expectedInternalConfig.lastObservedGeneration, tt.internalConfig.lastObservedGeneration)
			}
			if tt.internalConfig.desiredLogLevel != tt.expectedInternalConfig.desiredLogLevel {
				t.Errorf("unexpected 'desiredLogLevel' value; wanted=%v, got=%v", tt.expectedInternalConfig.desiredLogLevel, tt.internalConfig.desiredLogLevel)
			}

			config, err := client.OperatorV1alpha1().ClusterVersionOperators().Get(ctx, "", metav1.GetOptions{})
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			if diff := cmp.Diff(tt.expectedConfig, *config, cmpopts.IgnoreFields(metav1.ObjectMeta{}, "ManagedFields")); diff != "" {
				t.Errorf("unexpected config (-want, +got) = %v", diff)
			}

			// Shutdown created resources
			cancelFunc()
		})
	}
}

func TestClusterVersionOperatorConfiguration_Sync(t *testing.T) {
	tests := []struct {
		name           string
		config         *operatorv1alpha1.ClusterVersionOperator
		expectedConfig *operatorv1alpha1.ClusterVersionOperator
	}{
		{
			name:           "the configuration resource does not exist in the cluster -> ignore",
			config:         nil,
			expectedConfig: nil,
		},
		{
			name: "Sync updates the ClusterVersionOperator resource",
			config: &operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "cluster",
					Generation: 4,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.Trace,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 3,
				},
			},
			expectedConfig: &operatorv1alpha1.ClusterVersionOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "cluster",
					Generation: 4,
				},
				Spec: operatorv1alpha1.ClusterVersionOperatorSpec{
					OperatorLogLevel: operatorv1.Trace,
				},
				Status: operatorv1alpha1.ClusterVersionOperatorStatus{
					ObservedGeneration: 4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize testing logic
			var client *operatorclientsetfake.Clientset
			if tt.config != nil {
				client = operatorclientsetfake.NewClientset(tt.config)
			} else {
				client = operatorclientsetfake.NewClientset()
			}

			factory := operatorexternalversions.NewSharedInformerFactoryWithOptions(client, time.Minute)
			cvoConfiguration := NewClusterVersionOperatorConfiguration(client, factory)
			defer cvoConfiguration.queue.ShutDown()

			ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
			defer cancelFunc()

			err := cvoConfiguration.Start(ctx)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

			// Run tested functionality
			err = cvoConfiguration.Sync(ctx, "ClusterVersionOperator/cluster")
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

			// Verify results
			config, err := client.OperatorV1alpha1().ClusterVersionOperators().Get(ctx, "cluster", metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				t.Errorf("unexpected error %v", err)
			}

			switch {
			case apierrors.IsNotFound(err) && tt.expectedConfig != nil:
				t.Errorf("expected config to be '%v', got NotFound", *tt.expectedConfig)
			case err == nil:
				if diff := cmp.Diff(*tt.expectedConfig, *config, cmpopts.IgnoreFields(metav1.ObjectMeta{}, "ManagedFields")); diff != "" {
					t.Errorf("unexpected config (-want, +got) = %v", diff)
				}
			}
		})
	}
}
