package iampolicy

type Document interface {
    IsEqual(Document) (bool, error)
    Marshal() (string, error)
    Unmarshal(string) error
}

type Statement struct {
    Sid    string `json:",omitempty"`
    Effect string  // Allow/Deny
    Action []string `json:",omitempty"`  // this can also be a string
    NotAction []string `json:",omitempty"`  // this can also be a string
    Resource []string  // this can also be a string
    Condition map[string]interface{} `json:",omitempty"`
}

type document struct {
    Version string
    Statements []Statement `json:"Statement"`
}

func (d* document) Marshal() (string, error) {
    panic("implement me")
}

func (d* document) Unmarshal(s string) error {
    panic("implement me")
}

func (d* document) IsEqual(d2 Document) (bool, error) {
    panic("not implemented")
}

var _ Document = &document{}