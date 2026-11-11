package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOriginPatterns(t *testing.T) {
	patterns, err := Config{FrontOrigin: "http://localhost:5180"}.OriginPatterns()
	require.NoError(t, err)
	assert.Equal(t, []string{"localhost:5180"}, patterns)

	_, err = Config{FrontOrigin: "localhost"}.OriginPatterns()
	assert.Error(t, err, "no scheme: not a URL")
}

func TestRedactedDBURL(t *testing.T) {
	redacted := Config{DBURL: "postgres://donut:secret@localhost:5434/donut?search_path=donut"}.RedactedDBURL()
	assert.NotContains(t, redacted, "secret")
	assert.Contains(t, redacted, "localhost:5434/donut")
}
