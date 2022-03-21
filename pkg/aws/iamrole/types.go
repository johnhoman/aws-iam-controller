package iamrole

import (
	"time"
)

type AttachOptions struct {
	Name      string
	PolicyArn string
}

type DetachOptions = AttachOptions
type ListOptions = GetOptions

type CreateOptions struct {
	Name               string
	Description        string
	MaxDurationSeconds int32
	PolicyDocument     string
}

type GetOptions struct {
	Name string
}

type UpdateOptions struct {
	Name               string
	Description        string
	MaxDurationSeconds int32
	PolicyDocument     string
}

type DeleteOptions struct {
	Name string
}

type IamRole struct {
	Arn         string
	CreateDate  time.Time
	Description string
	Id          string
	Name        string
	TrustPolicy string
}

type AttachedPolicy struct {
	Name string
	Arn  string
}

type PolicyMap map[string]string

func (m PolicyMap) Contains(name string) bool {
	_, ok := m[name]
	return ok
}

func (m PolicyMap) Set(name string, arn string) {
	m[name] = arn
}

func (m PolicyMap) Get(name string) (string, bool) {
	v, ok := m[name]
	return v, ok
}

func (m PolicyMap) Delete(name string) bool {
	_, ok := m[name]
	if !ok {
		return false
	}
	delete(m, name)
	return true
}

type AttachedPolicies []AttachedPolicy

func (p *AttachedPolicies) Len() int {
	return len(*p)
}

func (p *AttachedPolicies) Insert(name, arn string) {
	policy := AttachedPolicy{Name: name, Arn: arn}
	*p = append(*p, policy)
}

func (p *AttachedPolicies) ToMap() PolicyMap {
	m := PolicyMap{}
	for _, item := range *p {
		m.Set(item.Name, item.Arn)
	}
	return m
}
