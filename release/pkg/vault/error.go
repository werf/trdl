package vault

import "regexp"

type ErrorHelper struct {
	RetriablePatterns []*regexp.Regexp
}

func NewErrorHelper(messages []string) *ErrorHelper {
	var patterns []*regexp.Regexp
	for _, msg := range messages {
		patterns = append(patterns, regexp.MustCompile("(?i)"+msg))
	}
	return &ErrorHelper{
		RetriablePatterns: patterns,
	}
}

func (h *ErrorHelper) isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, pattern := range h.RetriablePatterns {
		if pattern.MatchString(msg) {
			return true
		}
	}
	return false
}
