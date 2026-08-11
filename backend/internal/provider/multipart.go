package provider

import (
	"fmt"
	"net/textproto"
	"strings"
)

func makeTextprotoFileHeader(fieldName, filename, contentType string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader)
	header.Set(
		"Content-Disposition",
		fmt.Sprintf(
			`form-data; name="%s"; filename="%s"`,
			escapeMultipartValue(fieldName),
			escapeMultipartValue(filename),
		),
	)
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	header.Set("Content-Type", contentType)
	return header
}

func escapeMultipartValue(value string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, `\"`, "\r", "", "\n", "").Replace(value)
}
