package bindmanager

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"

	"github.com/johnhoman/aws-iam-controller/api/v1alpha1"
)

type Binding struct {
	Role            *v1alpha1.IamRole
	ServiceAccounts []corev1.ObjectReference
}

type condition struct {
	StringEquals map[string]interface{} `json:",omitempty"` // nolint: tagliatelle
}

type principal struct {
	AWS       interface{} `json:",omitempty"` // nolint: tagliatelle
	Federated string      `json:",omitempty"` // nolint: tagliatelle
}

type statement struct {
	Sid       string    `json:",omitempty"` // nolint: tagliatelle
	Effect    string    `json:",omitempty"` // nolint: tagliatelle
	Principal principal `json:",omitempty"` // nolint: tagliatelle
	Action    string    `json:",omitempty"` // nolint: tagliatelle
	Condition condition `json:",omitempty"` // nolint: tagliatelle
}

type policyDocument struct {
	Version    string
	Statements []statement `json:"Statement"` // nolint: tagliatelle
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
