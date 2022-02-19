package iampolicy

import "time"

type CreateOptions struct {
    Name string
    Document string
    Description string
}

type DeleteOptions struct {
    Arn string
}

type GetOptions struct {
    Arn string
    Name string
}

type UpdateOptions struct {
    Arn string
    Document string
}

type IamPolicy struct {
    Arn string
    AttachmentCount int32
    CreateDate *time.Time
    Document string
    Description string
    Name string
    Id string
}

