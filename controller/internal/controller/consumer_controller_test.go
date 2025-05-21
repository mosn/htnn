package controller

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

// test_plugins.go
// Test plugin A (simulated KeyAuth)
type TestPluginA struct{}

func (p *TestPluginA) Name() string                    { return "pluginA" }
func (p *TestPluginA) NewConfig() PluginConsumerConfig { return &TestConfigA{} }

type TestConfigA struct {
	Key string `json:"key"`
}

func (c *TestConfigA) Index() string { return c.Key }

// Test plugin B (simulated jwt)
type TestPluginB struct{}

func (p *TestPluginB) Name() string                    { return "pluginB" }
func (p *TestPluginB) NewConfig() PluginConsumerConfig { return &TestConfigB{} }

type TestConfigB struct {
	Issuer string `json:"issuer"`
}

func (c *TestConfigB) Index() string { return c.Issuer }

// consumer_reconciler_test.go
func TestConflictDetection(t *testing.T) {
	// Initialize the registry (Note: clear before testing to avoid contamination)
	pluginRegistry = make(map[string]Plugin)
	RegisterPlugin(&TestPluginA{})
	RegisterPlugin(&TestPluginB{})

	// Create a test Reconciler
	r := &ConsumerReconciler{
		keyIndex: NewKeyIndexRegistry(),
	}

	tests := []struct {
		name        string
		consumers   []*mosniov1.Consumer
		wantErr     bool
		errContains string
	}{
		{
			name: "no conflict with different plugins",
			consumers: []*mosniov1.Consumer{
				createTestConsumer("ns1", "consumer1", "pluginA", `{"key": "key1"}`),
				createTestConsumer("ns1", "consumer2", "pluginB", `{"issuer": "key1"}`),
			},
			wantErr: false,
		},
		{
			name: "conflict within same plugin",
			consumers: []*mosniov1.Consumer{
				createTestConsumer("ns1", "consumer1", "pluginA", `{"key": "dupKey"}`),
				createTestConsumer("ns1", "consumer2", "pluginA", `{"key": "dupKey"}`),
			},
			wantErr:     true,
			errContains: "key conflict",
		},
		{
			name: "unknown plugin should fail",
			consumers: []*mosniov1.Consumer{
				createTestConsumer("ns1", "consumer1", "nonExistPlugin", `{}`),
			},
			wantErr:     true,
			errContains: "unknown plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the index status
			r.keyIndex.index = make(map[string]map[string]map[string]string)

			// Simulate conflict detection in the Reconcile process
			var err error
			for _, c := range tt.consumers {
				if e := r.indexConsumer(c.Namespace, c); e != nil {
					err = e
					break
				}
			}

			// verification result
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Auxiliary function: Create a test Consumer object
func createTestConsumer(namespace, name, pluginName, rawConfig string) *mosniov1.Consumer {
	return &mosniov1.Consumer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: mosniov1.ConsumerSpec{
			Auth: map[string]mosniov1.ConsumerPlugin{
				pluginName: {
					Config: runtime.RawExtension{
						Raw: []byte(rawConfig),
					},
				},
			},
		},
	}
}
