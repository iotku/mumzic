package youtubedl

import (
	"bufio"
	_ "embed"
	"strings"
	"testing"
)

//go:embed whitelist-test.txt
var whitelistExample string

func TestIsWhiteListedUrl(t *testing.T) {
	scanURLPrefixes(bufio.NewScanner(strings.NewReader(whitelistExample)))
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://youtu.be/dQw4w9WgXcQ", true},
		{"https://notyoutube.com/watch?v=qwqwdqwd", false},
	}

	for _, tt := range tests {
		got := IsWhiteListedURL(tt.url)
		if got != tt.expected {
			t.Errorf("IsWhiteListedURL(%q) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}
