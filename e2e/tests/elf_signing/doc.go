// Package elf_signing holds the ELF-signing e2e suite. This file carries no
// build tag so the package is not fully excluded on unsupported platforms:
// ginkgo fails a package whose every Go file is build-constrained out ("build
// constraints exclude all Go files"), whereas one untagged file makes it skip.
package elf_signing
