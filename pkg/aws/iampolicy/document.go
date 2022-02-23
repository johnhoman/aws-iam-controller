package iampolicy

import (
    "k8s.io/apimachinery/pkg/util/json"
    "reflect"
)

type Document interface {
    Equals(Document) (bool, error)
    SetStatements([]Statement)
    GetStatements() []Statement
    GetVersion() string

    Marshal() (string, error)
}

type Statement struct {
    Sid    string `json:",omitempty"`
    Effect string  // Allow/Deny
    Action interface{} `json:",omitempty"`  // this can also be a string
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

func (d* document) Equals(d2 Document) (bool, error) {
    return reflect.DeepEqual(d, d2), nil
}

func (d *document) GetVersion() string {
    return d.Version
}

func (d *document) SetStatements(statements []Statement) {
    for k := 0; k < len(statements); k++ {
        statement := statements[k]
        if _, ok := statement.Action.(string); ok {
            statement.Action = []interface{}{statement.Action}
        }
        if _, ok := statement.Resource.(string); ok {
            statement.Resource = []interface{}{statement.Resource}
        }
        statements[k] = statement
    }
    d.Statements = statements
}

func (d *document) GetStatements() []Statement {
    return d.Statements
}

func (d* document) unmarshal(s string) error {
    raw := []byte(s)
    if err := json.Unmarshal(raw, d); err != nil {
        return err
    }
    d.SetStatements(d.GetStatements())
    return nil
}

func NewDocument() Document {
    return &document{Version: "2012-10-17"}
}

func NewDocumentFromString(doc string) (Document, error) {
    d := NewDocument().(*document)
    if err := d.unmarshal(doc); err != nil {
        return nil, err
    }
    return d, nil
}

var _ Document = &document{}