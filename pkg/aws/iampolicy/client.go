package iampolicy

import (
	"context"
)

type Client struct {}

func (c* Client) Create(ctx context.Context, options *CreateOptions) (*IamPolicy, error) {
    panic("implement me")
}

func (c* Client) Update(ctx context.Context, options *UpdateOptions) (*IamPolicy, error) {
    panic("implement me")
}

func (c* Client) Get(ctx context.Context, options *GetOptions) (*IamPolicy, error) {
    panic("implement me")
}

func (c* Client) Delete(ctx context.Context, options *DeleteOptions) error {
    panic("implement me")
}

var _ Interface = &Client{}