package git

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TODO: refactor this test
var _ = XDescribe("Signatures loader from git NOTES", func() {
	It("successfully loads werf signatures", func() {
		tagName := "v1.2.84+fix1"

		repo, err := CloneInMemory("https://github.com/werf/werf.git", CloneOptions{TagName: tagName})
		Expect(err).To(Succeed())

		tref, err := repo.Tag(tagName)
		Expect(err).To(Succeed())

		tobj, err := repo.TagObject(tref.Hash())
		Expect(err).To(Succeed())

		sigs, err := objectSignaturesFromNotes(repo, tobj.Hash.String())
		Expect(err).To(Succeed())

		Expect(len(sigs) > 0).To(BeTrue())
	})
})
