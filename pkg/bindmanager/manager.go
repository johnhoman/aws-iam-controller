package bindmanager

import (
	"context"
	"fmt"
	"reflect"
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
	sid := sidLabel(binding.Role.Name, binding.Role.Namespace)
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

	serviceAccounts := make([]string, 0, len(binding.ServiceAccounts))
	for _, name := range serviceAccounts {
		serviceAccounts = append(
			serviceAccounts,
			serviceAccountFormat(binding.GetNamespace(), name),
		)
	}
	// If there's no service account then remove the statement
	var accounts interface{} = serviceAccounts
	if len(serviceAccounts) == 1 {
		accounts = serviceAccounts[0]
	}

	stmt := statement{
		Sid:       sid,
		Effect:    EffectAllow,
		Principal: principal{Federated: b.oidcArn},
		Action:    ActionAssumeRoleWithWebIdentity,
		Condition: condition{
			StringEquals: map[string]interface{}{b.issuer: accounts},
		},
	}
	if found < 0 {
		// Not found
		if len(binding.ServiceAccounts) > 0 {
			doc.Statements = append(doc.Statements, stmt)
		}
	} else {
		// Found
		if len(binding.ServiceAccounts) == 0 {
			statements := make([]statement, 0, len(doc.Statements)-1)
			for k, stm := range doc.Statements {
				if k != found {
					statements = append(statements, stm)
				}
			}
			doc.Statements = statements
		} else {
			doc.Statements[found] = stmt
		}
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

func serviceAccountFormat(namespace, name string) string {
	return fmt.Sprintf(SubjectFormat, namespace, name)
}

func SidLabel(name, namespace string) string {
	return sidLabel(name, namespace)
}
