package vault

import "regexp"

var retriablePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)busy`),
	regexp.MustCompile(`(?i)not enough verified PGP signatures`),
}

func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, pattern := range retriablePatterns {
		if pattern.MatchString(msg) {
			return true
		}
	}
	return false
}
