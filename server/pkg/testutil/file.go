package testutil

import (
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
)

func CopyIn(sourcePath, destinationPath string) {
	Expect(copy.Copy(sourcePath, destinationPath)).Should(Succeed())
}
