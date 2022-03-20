package iamrole

import "time"

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
