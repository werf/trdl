// Package mac_signing holds the mac-signing e2e suite. This file carries no
// build tag so the package is not fully excluded when the ai_tests tag is
// off: ginkgo fails a package whose every Go file is build-constrained out
// ("build constraints exclude all Go files"), whereas one untagged file makes
// it skip.
package mac_signing
