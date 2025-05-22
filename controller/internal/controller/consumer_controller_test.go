package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
	"k8s.io/apimachinery/pkg/types"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/controller/internal/controller/component"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sync"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

type mockPlugin struct {
	name   string
	config api.PluginConsumerConfig
}

func (p *mockPlugin) Config() api.PluginConfig {
	return p.config.(api.PluginConfig)
}

func (p *mockPlugin) Type() plugins.PluginType {
	return plugins.TypeAuthn
}

func (p *mockPlugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{Position: plugins.OrderPositionAuthn}
}

func (p *mockPlugin) Merge(parent interface{}, child interface{}) interface{} {
	return nil
}

func (p *mockPlugin) NonBlockingPhases() api.Phase {
	return 0
}

func (p *mockPlugin) ConsumerConfig() api.PluginConsumerConfig {
	return p.config.(api.PluginConsumerConfig)
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

func init() {
	plugins.RegisterPluginType("testPlugin", &mockPlugin{
		name:   "testPlugin",
		config: &testPluginConfig{},
	})
}

func createTestConsumer(namespace, name, pluginName, key string) *mosniov1.Consumer {
	config := map[string]interface{}{"key": key}
	rawConfig, _ := json.Marshal(config)

	return &mosniov1.Consumer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
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
	}
}

func TestKeyIndexRegistry(t *testing.T) {
	registry := NewKeyIndexRegistry()

	// Test concurrent security
	t.Run("concurrent access", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				registry.mu.Lock()
				defer registry.mu.Unlock()
				// Analog write operation
				if registry.index["ns"] == nil {
					registry.index["ns"] = make(map[string]map[string]string)
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestIndexConsumer(t *testing.T) {
	r := &ConsumerReconciler{
		keyIndex: NewKeyIndexRegistry(),
	}

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
					r.keyIndex.index[consumer.Namespace]["testPlugin"]["key123"])
			}
		})
	}
}

func TestCheckConsumerConflicts(t *testing.T) {
	r := &ConsumerReconciler{
		keyIndex: NewKeyIndexRegistry(),
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

		err := r.checkConsumerConflicts(context.Background(), state)
		assert.NoError(t, err)
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

		err := r.checkConsumerConflicts(context.Background(), state)
		assert.NoError(t, err)

		// The consumer status of the verified conflict is updated
		assert.Equal(t, metav1.ConditionFalse, conflictConsumer.Status.Conditions[0].Status)
	})
}

func TestReconcile(t *testing.T) {
	fakeClient := fake.NewClientBuilder().
		WithLists(&mosniov1.ConsumerList{
			Items: []mosniov1.Consumer{
				*createTestConsumer("ns1", "valid", "testPlugin", "key1"),
				*createTestConsumer("ns1", "invalid", "testPlugin", "{broken}"),
			},
		}).
		Build()

	r := &ConsumerReconciler{
		ResourceManager: component.NewK8sResourceManager(fakeClient),
		keyIndex:        NewKeyIndexRegistry(),
	}

	t.Run("handle mixed consumers", func(t *testing.T) {
		result, err := r.Reconcile(context.Background(), ctrl.Request{})
		assert.NoError(t, err)
		assert.False(t, result.Requeue)

		// Verify that the status update is invalid consumer
		var invalidConsumer mosniov1.Consumer
		_ = fakeClient.Get(context.Background(),
			types.NamespacedName{Namespace: "ns1", Name: "invalid"},
			&invalidConsumer)
		assert.Equal(t, metav1.ConditionFalse, invalidConsumer.Status.Conditions[0].Status)
	})
}
