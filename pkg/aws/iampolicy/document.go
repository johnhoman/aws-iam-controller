/*
Copyright 2022 John Homan

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package iampolicy

import (
	"k8s.io/apimachinery/pkg/util/json"
	"reflect"
)

type Document interface {
	Equals(Document) (bool, error)
	SetStatements([]Statement)
	GetStatements() []Statement
	SetVersion(string)
	GetVersion() string
	Marshal() (string, error)
}

type Conditions struct {
	ArnLike                           map[string][]string `json:",omitempty"` // nolint: tagliatelle
	ArnLikeIfExists                   map[string][]string `json:",omitempty"` // nolint: tagliatelle
	ArnNotLike                        map[string][]string `json:",omitempty"` // nolint: tagliatelle
	ArnNotLikeIfExists                map[string][]string `json:",omitempty"` // nolint: tagliatelle
	BinaryEquals                      map[string][]string `json:",omitempty"` // nolint: tagliatelle
	BinaryEqualsIfExists              map[string][]string `json:",omitempty"` // nolint: tagliatelle
	Bool                              map[string][]string `json:",omitempty"` // nolint: tagliatelle
	BoolIfExists                      map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateEquals                        map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateEqualsIfExists                map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateNotEquals                     map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateNotEqualsIfExists             map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateLessThan                      map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateLessThanIfExists              map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateLessThanEquals                map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateLessThanEqualsIfExists        map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateGreaterThan                   map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateGreaterThanIfExists           map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateGreaterThanEquals             map[string][]string `json:",omitempty"` // nolint: tagliatelle
	DateGreaterThanEqualsIfExists     map[string][]string `json:",omitempty"` // nolint: tagliatelle
	IpAddress                         map[string][]string `json:",omitempty"` // nolint: tagliatelle
	IpAddressIfExists                 map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NotIpAddress                      map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NotIpAddressIfExists              map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericEquals                     map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericEqualsIfExists             map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericNotEquals                  map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericNotEqualsIfExists          map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericLessThan                   map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericLessThanIfExists           map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericLessThanEquals             map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericLessThanEqualsIfExists     map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericGreaterThan                map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericGreaterThanIfExists        map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericGreaterThanEquals          map[string][]string `json:",omitempty"` // nolint: tagliatelle
	NumericGreaterThanEqualsIfExists  map[string][]string `json:",omitempty"` // nolint: tagliatelle
	Null                              map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringLike                        map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringLikeIfExists                map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringNotLike                     map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringNotLikeIfExists             map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringEquals                      map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringEqualsIfExists              map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringNotEquals                   map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringNotEqualsIfExists           map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringEqualsIgnoreCase            map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringEqualsIgnoreCaseIfExists    map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringNotEqualsIgnoreCase         map[string][]string `json:",omitempty"` // nolint: tagliatelle
	StringNotEqualsIgnoreCaseIfExists map[string][]string `json:",omitempty"` // nolint: tagliatelle
}

type Statement struct {
	Sid        string      `json:",omitempty"` // nolint: tagliatelle
	Effect     string      // Allow/Deny
	Action     interface{} `json:",omitempty"` // nolint: tagliatelle
	Resource   interface{}
	Conditions *Conditions `json:"Condition,omitempty"` // nolint: tagliatelle
}

type document struct {
	Version    string
	Statements []Statement `json:"Statement"` // nolint: tagliatelle
}

func (d *document) Marshal() (string, error) {
	raw, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (d *document) Equals(d2 Document) (bool, error) {
	return reflect.DeepEqual(d, d2), nil
}

func (d *document) GetVersion() string {
	return d.Version
}

func (d *document) SetVersion(s string) {
	d.Version = s
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

func (d *document) unmarshal(s string) error {
	raw := []byte(s)
	if err := json.Unmarshal(raw, d); err != nil {
		return err
	}
	d.SetStatements(d.GetStatements())
	return nil
}

func NewDocument() *document {
	return &document{Version: "2012-10-17"}
}

func NewDocumentFromString(doc string) (*document, error) {
	d := NewDocument()
	if err := d.unmarshal(doc); err != nil {
		return nil, err
	}
	return d, nil
}

var _ Document = &document{}
