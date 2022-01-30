package bindmanager

import (
	"context"

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
	iamPatch := client.MergeFrom(p.binding.ServiceAccount.DeepCopy())
	annotations := p.binding.ServiceAccount.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[IamRoleArnAnnotation] = p.binding.Role.Status.RoleArn
	p.binding.ServiceAccount.SetAnnotations(annotations)
	if err := c.Patch(ctx, p.binding.ServiceAccount, iamPatch); err != nil {
		return err
	}
	return nil
}

func (p *patch) Undo(ctx context.Context, client client.Client) error {
	panic("implement me")
}

var _ Patch = &patch{}
