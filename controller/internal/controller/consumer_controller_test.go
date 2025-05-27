package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

type mockPlugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *mockPlugin) Config() api.PluginConfig {
	return &testPluginConfig{}
}

func (p *mockPlugin) Type() plugins.PluginType {
	return plugins.TypeAuthn
}

func (p *mockPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{Position: plugins.OrderPositionAuthn}
}

func (p *mockPlugin) Factory() api.FilterFactory {
	return func(interface{}, api.FilterCallbackHandler) api.Filter {
		return &api.PassThroughFilter{}
	}
}

func (p *mockPlugin) NonBlockingPhases() api.Phase {
	return 0
}

type testPluginConfig struct {
	Key string `json:"key"`
}

func (c *testPluginConfig) ProtoReflect() protoreflect.Message {
	return nil
}

func (c *testPluginConfig) Validate() error {
	if c.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	return nil
}

func (c *testPluginConfig) Index() string {
	return c.Key
}

func (c *testPluginConfig) ConsumerConfig() interface{} {
	return c
}

func createTestConsumer(namespace, name, pluginName, key string) *mosniov1.Consumer {
	config := map[string]interface{}{"key": key}
	rawConfig, _ := json.Marshal(config)

	return &mosniov1.Consumer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       namespace,
			Name:            name,
			ResourceVersion: "1",
		},
		Spec: mosniov1.ConsumerSpec{
			Auth: map[string]mosniov1.ConsumerPlugin{
				pluginName: {
					Config: runtime.RawExtension{
						Raw: rawConfig,
					},
				},
			},
		},
		Status: mosniov1.ConsumerStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Accepted",
					Status: metav1.ConditionTrue,
				},
			},
		},
	}
}

func TestIndexConsumer(t *testing.T) {
	r := &ConsumerReconciler{
		KeyIndex: NewKeyIndexRegistry(),
	}

	plugins.RegisterPlugin("testPlugin", &mockPlugin{})

	plugin := plugins.LoadPluginType("testPlugin")
	assert.NotNil(t, plugin)

	tests := []struct {
		name     string
		setup    func() *mosniov1.Consumer
		wantErr  bool
		errMatch string
	}{
		{
			name: "successful index",
			setup: func() *mosniov1.Consumer {
				return createTestConsumer("ns1", "consumer1", "testPlugin", "key123")
			},
			wantErr: false,
		},
		{
			name: "conflict detection",
			setup: func() *mosniov1.Consumer {
				c1 := createTestConsumer("ns1", "consumer1", "testPlugin", "dupKey")
				_ = r.indexConsumer("ns1", c1)

				return createTestConsumer("ns1", "consumer2", "testPlugin", "dupKey")
			},
			wantErr:  true,
			errMatch: "key conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := tt.setup()
			err := r.indexConsumer(consumer.Namespace, consumer)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMatch)
			} else {
				assert.NoError(t, err)
				// Verify that the index is correctly added
				assert.Equal(t,
					consumer.Name,
					r.KeyIndex.index[consumer.Namespace]["testPlugin"]["key123"])
			}
		})
	}
}

func TestCheckConsumerConflicts(t *testing.T) {
	r := &ConsumerReconciler{
		KeyIndex: NewKeyIndexRegistry(),
	}

	t.Run("multiple namespaces no conflict", func(t *testing.T) {
		state := &consumerReconcileState{
			namespaceToConsumers: map[string]map[string]*mosniov1.Consumer{
				"ns1": {
					"consumer1": createTestConsumer("ns1", "consumer1", "testPlugin", "key1"),
				},
				"ns2": {
					"consumer2": createTestConsumer("ns2", "consumer2", "testPlugin", "key1"),
				},
			},
		}

		r.checkConsumerConflicts(context.Background(), state)

	})

	t.Run("detect cross-consumer conflict", func(t *testing.T) {
		conflictConsumer := createTestConsumer("ns1", "consumer2", "testPlugin", "dupKey")
		state := &consumerReconcileState{
			namespaceToConsumers: map[string]map[string]*mosniov1.Consumer{
				"ns1": {
					"consumer1": createTestConsumer("ns1", "consumer1", "testPlugin", "dupKey"),
					"consumer2": conflictConsumer,
				},
			},
		}

		r.checkConsumerConflicts(context.Background(), state)

		// The consumer status of the verified conflict is updated
		assert.Equal(t, metav1.ConditionFalse, conflictConsumer.Status.Conditions[0].Status)
	})
}
