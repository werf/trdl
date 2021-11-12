package gendocs

type PagesGenerator interface {
	HandlePath(pathPattern string, doc []byte) error
	Close() error
	HasFormatPathLink() bool
	FormatPathLink(pathPattern string) string
}
