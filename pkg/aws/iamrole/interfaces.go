package iamrole

import "context"

type Interface interface {
	Create(ctx context.Context, options *CreateOptions) (*IamRole, error)
	Update(ctx context.Context, options *UpdateOptions) (*IamRole, error)
	Get(ctx context.Context, options *GetOptions) (*IamRole, error)
	Delete(ctx context.Context, options *DeleteOptions) error
}
