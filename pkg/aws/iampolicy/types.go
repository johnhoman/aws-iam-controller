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
    CreateDate time.Time
    Document string
    Description string
    // VersionId the id of the default version. This client will only maintain
    // a single version -- all others will be deleted after being updated
    VersionId string
    Name string
    Id string
}

