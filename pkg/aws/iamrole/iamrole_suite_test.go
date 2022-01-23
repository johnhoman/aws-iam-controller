package iamrole_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestIamrole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Iamrole Suite")
}
