package iampolicy

import (
    "context"
)

type CreateOptions struct {}
type DeleteOptions struct {}
type GetOptions struct {}
type UpdateOptions struct {}
type IamPolicy struct {}

type Interface interface {
    Create(ctx context.Context, options *CreateOptions) (*IamPolicy, error)
    Update(ctx context.Context, options *UpdateOptions) (*IamPolicy, error)
    Get(ctx context.Context, options *GetOptions) (*IamPolicy, error)
    Delete(ctx context.Context, options *DeleteOptions) error
}