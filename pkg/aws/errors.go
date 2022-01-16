package aws

import (
	"errors"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func IsNotFound(err error) bool {
	oe := &iamtypes.NoSuchEntityException{}
	return errors.As(err, &oe)
}
