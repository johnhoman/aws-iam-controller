package iampolicy_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var ctx = context.Background()

func TestIampolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Iampolicy Suite")
}
