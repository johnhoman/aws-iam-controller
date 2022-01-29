package bindmanager

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Patch interface {
	Do(ctx context.Context, client client.Client) error
	Undo(ctx context.Context, client client.Client) error
}

type Manager interface {
	Bind(ctx context.Context, binding *Binding) error
	IsBound(ctx context.Context, binding *Binding) (bool, error)
	Patch(binding *Binding, options ...client.PatchOption) Patch
}
