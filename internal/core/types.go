package core

type ContentType string

const (
	ContentTypeText    ContentType = "text"
	ContentTypeURL     ContentType = "url"
	ContentTypeCommand ContentType = "command"
	ContentTypeCode    ContentType = "code"
)
