package publisher

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPublisher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Publisher suite")
}
