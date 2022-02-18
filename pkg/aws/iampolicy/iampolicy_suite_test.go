package iampolicy_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestIampolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Iampolicy Suite")
}
