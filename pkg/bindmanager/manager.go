package bindmanager

import (
	"context"
	"fmt"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"github.com/johnhoman/aws-iam-controller/pkg/aws/iamrole"
)

const (
	EffectAllow                     = "Allow"
	ActionAssumeRoleWithWebIdentity = "sts:AssumeRoleWithWebIdentity"
	SidLabelFormat                  = "AllowServiceAccount-%s-%s"
	SubjectFormat                   = "system:serviceaccount:%s:%s"
)

type bindManager struct {
	iamrole.Interface
	oidcArn string
	issuer  string
}

// Bind will establish a trust relationship between a role and a service account
// by allowing the service account to AssumeRoleWithWebIdentity
func (b *bindManager) Bind(ctx context.Context, binding *Binding) error {
	sid := sidLabel(binding.ServiceAccount.Name, binding.ServiceAccount.Namespace)
	upstream, err := b.Get(ctx, &iamrole.GetOptions{Name: binding.Role.GetName()})
	if err != nil {
		return err
	}
	original := &policyDocument{}
	doc := &policyDocument{}
	if err := doc.Unmarshal(upstream.TrustPolicy); err != nil {
		return err
	}
	*original = *doc
	found := -1
	for k, st := range doc.Statements {
		if strings.Compare(st.Sid, sid) == 0 {
			found = k
		}
	}
	stmt := statement{
		Sid:       sid,
		Effect:    EffectAllow,
		Principal: principal{Federated: b.oidcArn},
		Action:    ActionAssumeRoleWithWebIdentity,
		Condition: condition{
			StringEquals: map[string]interface{}{
				b.issuer: fmt.Sprintf(
					SubjectFormat,
					binding.ServiceAccount.GetNamespace(),
					binding.ServiceAccount.GetName(),
				),
			},
		},
	}
	if found < 0 {
		doc.Statements = append(doc.Statements, stmt)
	} else {
		doc.Statements[found] = stmt
	}
	if !reflect.DeepEqual(doc, original) {
		trust, err := doc.Marshal()
		if err != nil {
			return err
		}
		if _, err := b.Update(ctx, &iamrole.UpdateOptions{
			Name:           binding.Role.GetName(),
			PolicyDocument: trust,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *bindManager) Unbind(ctx context.Context, binding *Binding) error {
	upstream, err := b.Get(ctx, &iamrole.GetOptions{Name: binding.Role.GetName()})
	if err != nil {
		return err
	}
	doc := &policyDocument{}
	if err := doc.Unmarshal(upstream.TrustPolicy); err != nil {
		return err
	}
	sid := sidLabel(binding.ServiceAccount.Name, binding.ServiceAccount.Namespace)

	statements := make([]statement, 0, len(doc.Statements)-1)
	for _, stmt := range doc.Statements {
		if stmt.Sid != sid {
			statements = append(statements, stmt)
		}
	}
	if len(statements) < len(doc.Statements) {
		doc.Statements = statements
		trust, err := doc.Marshal()
		if err != nil {
			return err
		}
		_, err = b.Update(ctx, &iamrole.UpdateOptions{
			Name:           binding.Role.GetName(),
			PolicyDocument: trust,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// IsBound returns true if the expected statement id for the service account
// existing in the iam role's trust policy
func (b *bindManager) IsBound(ctx context.Context, binding *Binding) (bool, error) {
	upstream, err := b.Get(ctx, &iamrole.GetOptions{Name: binding.Role.GetName()})
	if err != nil {
		return false, err
	}
	doc := &policyDocument{}
	if err := doc.Unmarshal(upstream.TrustPolicy); err != nil {
		return false, err
	}
	sid := sidLabel(binding.ServiceAccount.Name, binding.ServiceAccount.Namespace)
	for _, stmt := range doc.Statements {
		if stmt.Sid == sid {
			return true, nil
		}
	}
	return false, nil
}

func (b *bindManager) Patch(binding *Binding, options ...client.PatchOption) Patch {
	return &patch{binding: binding, options: options}
}

var _ Manager = &bindManager{}

// New returns a new BindManager instance
func New(p iamrole.Interface, oidcArn string) *bindManager {
	issuer := oidcArn[strings.Index(oidcArn, "/")+1:] + ":sub"
	return &bindManager{Interface: p, oidcArn: oidcArn, issuer: issuer}
}

func sidLabel(name, namespace string) string {
	sid := fmt.Sprintf(SidLabelFormat, namespace, name)
	sid = strings.ReplaceAll(sid, "-", " ")
	sid = strings.Title(sid)
	sid = strings.ReplaceAll(sid, " ", "")
	return sid
}

func SidLabel(name, namespace string) string {
	return sidLabel(name, namespace)
}
