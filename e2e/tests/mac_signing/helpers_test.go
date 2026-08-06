package mac_signing

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/werf/trdl/server/pkg/testutil"
)

func composePort(service, containerPort string) string {
	hostPort := testutil.SucceedCommandOutputString(testDir, "docker", "compose", "port", service, containerPort)

	return strings.TrimSpace(hostPort[strings.LastIndex(hostPort, ":")+1:])
}

func setEnv(name, value string) {
	previous, existed := os.LookupEnv(name)
	Expect(os.Setenv(name, value)).To(Succeed())

	DeferCleanup(func() {
		if !existed {
			Expect(os.Unsetenv(name)).To(Succeed())
			return
		}
		Expect(os.Setenv(name, previous)).To(Succeed())
	})
}
