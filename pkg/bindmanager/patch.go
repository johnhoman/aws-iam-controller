package bindmanager

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IamRoleArnAnnotation = "eks.amazonaws.com/role-arn"
)

type patch struct {
	binding *Binding
	options []client.PatchOption
}

func (p *patch) Do(ctx context.Context, c client.Client) error {
	saPatch := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				IamRoleArnAnnotation: p.binding.Role.Status.RoleArn,
			},
		},
	}}
	saPatch.SetName(p.binding.ServiceAccount.GetName())
	saPatch.SetGroupVersionKind(p.binding.ServiceAccount.GroupVersionKind())
	if err := c.Patch(ctx, saPatch, client.Apply, p.options...); err != nil {
		return err
	}
	return nil
}

func (p *patch) Undo(ctx context.Context, client client.Client) error {
	panic("implement me")
}

var _ Patch = &patch{}
