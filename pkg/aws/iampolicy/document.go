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
	ArnLike                           map[string][]string `json:",omitempty"`
	ArnLikeIfExists                   map[string][]string `json:",omitempty"`
	ArnNotLike                        map[string][]string `json:",omitempty"`
	ArnNotLikeIfExists                map[string][]string `json:",omitempty"`
	BinaryEquals                      map[string][]string `json:",omitempty"`
	BinaryEqualsIfExists              map[string][]string `json:",omitempty"`
	Bool                              map[string][]string `json:",omitempty"`
	BoolIfExists                      map[string][]string `json:",omitempty"`
	DateEquals                        map[string][]string `json:",omitempty"`
	DateEqualsIfExists                map[string][]string `json:",omitempty"`
	DateNotEquals                     map[string][]string `json:",omitempty"`
	DateNotEqualsIfExists             map[string][]string `json:",omitempty"`
	DateLessThan                      map[string][]string `json:",omitempty"`
	DateLessThanIfExists              map[string][]string `json:",omitempty"`
	DateLessThanEquals                map[string][]string `json:",omitempty"`
	DateLessThanEqualsIfExists        map[string][]string `json:",omitempty"`
	DateGreaterThan                   map[string][]string `json:",omitempty"`
	DateGreaterThanIfExists           map[string][]string `json:",omitempty"`
	DateGreaterThanEquals             map[string][]string `json:",omitempty"`
	DateGreaterThanEqualsIfExists     map[string][]string `json:",omitempty"`
	IpAddress                         map[string][]string `json:",omitempty"`
	IpAddressIfExists                 map[string][]string `json:",omitempty"`
	NotIpAddress                      map[string][]string `json:",omitempty"`
	NotIpAddressIfExists              map[string][]string `json:",omitempty"`
	NumericEquals                     map[string][]string `json:",omitempty"`
	NumericEqualsIfExists             map[string][]string `json:",omitempty"`
	NumericNotEquals                  map[string][]string `json:",omitempty"`
	NumericNotEqualsIfExists          map[string][]string `json:",omitempty"`
	NumericLessThan                   map[string][]string `json:",omitempty"`
	NumericLessThanIfExists           map[string][]string `json:",omitempty"`
	NumericLessThanEquals             map[string][]string `json:",omitempty"`
	NumericLessThanEqualsIfExists     map[string][]string `json:",omitempty"`
	NumericGreaterThan                map[string][]string `json:",omitempty"`
	NumericGreaterThanIfExists        map[string][]string `json:",omitempty"`
	NumericGreaterThanEquals          map[string][]string `json:",omitempty"`
	NumericGreaterThanEqualsIfExists  map[string][]string `json:",omitempty"`
	Null                              map[string][]string `json:",omitempty"`
	StringLike                        map[string][]string `json:",omitempty"`
	StringLikeIfExists                map[string][]string `json:",omitempty"`
	StringNotLike                     map[string][]string `json:",omitempty"`
	StringNotLikeIfExists             map[string][]string `json:",omitempty"`
	StringEquals                      map[string][]string `json:",omitempty"`
	StringEqualsIfExists              map[string][]string `json:",omitempty"`
	StringNotEquals                   map[string][]string `json:",omitempty"`
	StringNotEqualsIfExists           map[string][]string `json:",omitempty"`
	StringEqualsIgnoreCase            map[string][]string `json:",omitempty"`
	StringEqualsIgnoreCaseIfExists    map[string][]string `json:",omitempty"`
	StringNotEqualsIgnoreCase         map[string][]string `json:",omitempty"`
	StringNotEqualsIgnoreCaseIfExists map[string][]string `json:",omitempty"`
}

type Statement struct {
	Sid        string      `json:",omitempty"`
	Effect     string      // Allow/Deny
	Action     interface{} `json:",omitempty"` // this can also be a string
	Resource   interface{} // this can also be a string
	Conditions *Conditions `json:"Condition,omitempty"`
}

type document struct {
	Version    string
	Statements []Statement `json:"Statement"`
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
