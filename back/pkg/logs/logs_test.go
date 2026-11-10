package logs_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"donut/pkg/logs"
)

func TestNew(t *testing.T) {
	var out bytes.Buffer
	log, err := logs.New("info", &out)
	require.NoError(t, err)
	log.Debug("hidden")
	log.Info("shown", "player", "Ann\nfake line")
	assert.NotContains(t, out.String(), "hidden")
	assert.Contains(t, out.String(), `player="Ann\nfake line"`, "values are quoted: a name can't forge a log line")

	for _, level := range []string{"debug", "DEBUG", " warn ", "error"} {
		_, err := logs.New(level, &out)
		assert.NoError(t, err, level)
	}
	_, err = logs.New("loud", &out)
	assert.Error(t, err)
}
