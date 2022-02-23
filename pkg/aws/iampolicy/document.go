package iampolicy

import "k8s.io/apimachinery/pkg/util/json"

type Document interface {
    IsEqual(Document) (bool, error)
    Marshal() (string, error)
    Unmarshal(string) error
    AddStatement(...Statement)
    GetStatements() []Statement
    GetVersion() string
}

type Statement struct {
    Sid    string `json:",omitempty"`
    Effect string  // Allow/Deny
    Action interface{} `json:",omitempty"`  // this can also be a string
    NotAction interface{} `json:",omitempty"`  // this can also be a string
    Resource interface{}  // this can also be a string
    Condition map[string]interface{} `json:",omitempty"`
}

type document struct {
    Version string
    Statements []Statement `json:"Statement"`
}

func (d* document) Marshal() (string, error) {
    raw, err := json.Marshal(d)
    if err != nil {
        return "", err
    }
    return string(raw), nil
}

func (d* document) Unmarshal(s string) error {
    raw := []byte(s)
    if err := json.Unmarshal(raw, d); err != nil {
        return err
    }
    return nil
}

func (d* document) IsEqual(d2 Document) (bool, error) {
    panic("not implemented")
}

func (d *document) GetVersion() string {
    return d.Version
}

func (d *document) AddStatement(statements ...Statement) {
    d.Statements = append(d.Statements, statements...)
}

func (d *document) GetStatements() []Statement {
    return d.Statements
}

func NewDocument() Document {
    return &document{Version: "2012-10-17"}
}

var _ Document = &document{}