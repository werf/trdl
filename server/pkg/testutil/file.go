package testutil

import (
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
)

func CopyIn(sourcePath, destinationPath string) {
	Ω(copy.Copy(sourcePath, destinationPath)).Should(Succeed())
}
