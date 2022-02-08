package bindmanager

import (
	"k8s.io/apimachinery/pkg/util/json"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
)

type Binding struct {
	Role            *v1alpha1.IamRole
	ServiceAccounts []string
}

func (b *Binding) GetNamespace() string {
	return b.Role.GetNamespace()
}

type condition struct {
	StringEquals map[string]interface{} `json:",omitempty"`
}

type principal struct {
	AWS       interface{} `json:",omitempty"`
	Federated string      `json:",omitempty"`
}

type statement struct {
	Sid       string    `json:",omitempty"`
	Effect    string    `json:",omitempty"`
	Principal principal `json:",omitempty"`
	Action    string    `json:",omitempty"`
	Condition condition `json:",omitempty"`
}

type policyDocument struct {
	Version    string
	Statements []statement `json:"Statement"`
}

func (pd *policyDocument) Marshal() (string, error) {
	raw, err := json.Marshal(pd)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (pd *policyDocument) Unmarshal(doc string) error {
	b := []byte(doc)
	if err := json.Unmarshal(b, pd); err != nil {
		return err
	}
	return nil
}
