package carddav

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestSanitizeInlinePhoto(t *testing.T) {
	png := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 16)))
	tooBig := base64.StdEncoding.EncodeToString(append([]byte("\xff\xd8\xff\xe0"), make([]byte, maxInlinePhotoBytes)...))

	tests := []struct {
		name     string
		data     string
		wantData string
		wantType string
	}{
		{"jpeg", "/9j/4AAQ", "/9j/4AAQ", "image/jpeg"},
		{"png sniffed regardless of declared type", png, png, "image/png"},
		{"folded whitespace removed", "/9j/\r\n 4AAQ", "/9j/4AAQ", "image/jpeg"},
		{"not base64", "!!not-base64!!", "", ""},
		{"not an image", base64.StdEncoding.EncodeToString([]byte("<html><script>")), "", ""},
		{"over cap", tooBig, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, mediaType := sanitizeInlinePhoto(tt.data)
			if data != tt.wantData || mediaType != tt.wantType {
				t.Errorf("got (%.20q, %q), want (%.20q, %q)", data, mediaType, tt.wantData, tt.wantType)
			}
		})
	}
}
