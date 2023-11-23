package controller

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mockReader struct{}

func (r *mockReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return nil
}
func (r *mockReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

var _ client.Reader = &mockReader{}
