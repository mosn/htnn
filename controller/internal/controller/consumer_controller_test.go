package controller

import (
	"context"
	"encoding/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func TestCheckConsumerConflicts(t *testing.T) {
	scheme := runtime.NewScheme()
	mosniov1.AddToScheme(scheme)

	tests := []struct {
		name    string
		setup   func() []mosniov1.Consumer
		wantErr bool
	}{
		{
			name: "no consumers",
			setup: func() []mosniov1.Consumer {
				return []mosniov1.Consumer{}
			},
			wantErr: false,
		},
		{
			name: "single consumer no conflict",
			setup: func() []mosniov1.Consumer {
				return []mosniov1.Consumer{
					createTestConsumer("ns1", "consumer1", map[string]interface{}{
						"key": "value1",
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "multiple consumers different namespaces no conflict",
			setup: func() []mosniov1.Consumer {
				return []mosniov1.Consumer{
					createTestConsumer("ns1", "consumer1", map[string]interface{}{
						"key": "value1",
					}),
					createTestConsumer("ns2", "consumer2", map[string]interface{}{
						"key": "value1",
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "multiple consumers same namespace different plugins no conflict",
			setup: func() []mosniov1.Consumer {
				return []mosniov1.Consumer{
					createTestConsumerWithPlugins("ns1", "consumer1", map[string]map[string]interface{}{
						"plugin1": {"key": "value1"},
					}),
					createTestConsumerWithPlugins("ns1", "consumer2", map[string]map[string]interface{}{
						"plugin2": {"key": "value1"},
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "conflict in same namespace and plugin",
			setup: func() []mosniov1.Consumer {
				return []mosniov1.Consumer{
					createTestConsumer("ns1", "consumer1", map[string]interface{}{
						"key": "value1",
					}),
					createTestConsumer("ns1", "consumer2", map[string]interface{}{
						"key": "value1",
					}),
				}
			},
			wantErr: true,
		},
		{
			name: "invalid config",
			setup: func() []mosniov1.Consumer {
				return []mosniov1.Consumer{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "consumer1",
							Namespace: "ns1",
						},
						Spec: mosniov1.ConsumerSpec{
							Auth: map[string]mosniov1.ConsumerPlugin{
								"plugin1": {
									Config: runtime.RawExtension{
										Raw: []byte("invalid json"),
									},
								},
							},
						},
					},
				}
			},
			wantErr: true,
		},
		{
			name: "no key field",
			setup: func() []mosniov1.Consumer {
				return []mosniov1.Consumer{
					createTestConsumer("ns1", "consumer1", map[string]interface{}{
						"other_field": "value",
					}),
				}
			},
			wantErr: true,
		},
		{
			name: "issuer field as key",
			setup: func() []mosniov1.Consumer {
				return []mosniov1.Consumer{
					createTestConsumer("ns1", "consumer1", map[string]interface{}{
						"issuer": "issuer-value",
					}),
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumers := tt.setup()
			objects := make([]client.Object, 0, len(consumers))
			for i := range consumers {
				objects = append(objects, &consumers[i])
			}

			r := &ConsumerReconciler{
				ResourceManager: newTestResourceManager(scheme, objects...),
				keyIndex:        NewKeyIndexRegistry(),
			}

			err := r.checkConsumerConflicts(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("checkConsumerConflicts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func createTestConsumer(namespace, name string, config map[string]interface{}) mosniov1.Consumer {
	return createTestConsumerWithPlugins(namespace, name, map[string]map[string]interface{}{
		"test-plugin": config,
	})
}

func createTestConsumerWithPlugins(namespace, name string, plugins map[string]map[string]interface{}) mosniov1.Consumer {
	authPlugins := make(map[string]mosniov1.ConsumerPlugin)
	for pluginName, config := range plugins {
		raw, _ := json.Marshal(config)
		authPlugins[pluginName] = mosniov1.ConsumerPlugin{
			Config: runtime.RawExtension{Raw: raw},
		}
	}

	return mosniov1.Consumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: mosniov1.ConsumerSpec{
			Auth: authPlugins,
		},
	}
}

type testResourceManager struct {
	client  client.Client
	objects []client.Object
}

func (m *testResourceManager) Get(ctx context.Context, key client.ObjectKey, out client.Object) error {
	return m.client.Get(ctx, key, out)
}

func (m *testResourceManager) List(ctx context.Context, list client.ObjectList) error {
	return m.client.List(ctx, list)
}

func (m *testResourceManager) UpdateStatus(ctx context.Context, obj client.Object, statusPtr any) error {
	return nil
}

func newTestResourceManager(scheme *runtime.Scheme, objects ...client.Object) *testResourceManager {
	return &testResourceManager{
		client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objects...).
			Build(),
		objects: objects,
	}
}
