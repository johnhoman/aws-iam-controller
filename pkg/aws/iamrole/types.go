package iamrole

import "time"

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
