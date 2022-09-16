package git

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	It("objectFanoutPaths", func() {
		path := "44344dae578b8c9f53617f9dffec40b3f2ad91ae"
		expectedRes := []string{
			"44344dae578b8c9f53617f9dffec40b3f2ad91ae",
			"44/344dae578b8c9f53617f9dffec40b3f2ad91ae",
			"44/34/4dae578b8c9f53617f9dffec40b3f2ad91ae",
			"44/34/4d/ae578b8c9f53617f9dffec40b3f2ad91ae",
			"44/34/4d/ae/578b8c9f53617f9dffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b8c9f53617f9dffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c9f53617f9dffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f53617f9dffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53617f9dffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/617f9dffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f9dffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9dffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9d/ffec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9d/ff/ec40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9d/ff/ec/40b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9d/ff/ec/40/b3f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9d/ff/ec/40/b3/f2ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9d/ff/ec/40/b3/f2/ad91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9d/ff/ec/40/b3/f2/ad/91ae",
			"44/34/4d/ae/57/8b/8c/9f/53/61/7f/9d/ff/ec/40/b3/f2/ad/91/ae",
		}

		Expect(objectFanoutPaths(path)).To(Equal(expectedRes))
	})
})
