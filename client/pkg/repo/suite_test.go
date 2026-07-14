package repo

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Test_Suite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Repo Suite")
}
