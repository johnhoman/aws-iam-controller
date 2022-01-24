package bindmanager

import (
	"context"
)

type Manager interface {
	Bind(ctx context.Context, binding *Binding) error
	IsBound(ctx context.Context, binding *Binding) (bool, error)
}
