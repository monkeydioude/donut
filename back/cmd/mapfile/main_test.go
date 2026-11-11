package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"donut/pkg/logs"
)

// ADR 0007 test 18
func TestGenerateRefusesToOverwriteWithoutForce(t *testing.T) {
	maps := t.TempDir()
	folder := filepath.Join(maps, "world")
	require.NoError(t, run([]string{"city", "-out", filepath.Join(maps, "doma-1")}, logs.Discard()), "the city its dome leads to")
	require.NoError(t, run([]string{"generate", "-seed", "7", "-out", folder}, logs.Discard()), "a new folder")
	require.NoError(t, os.WriteFile(filepath.Join(folder, "map.json"), []byte("hand edit"), 0o644))

	err := run([]string{"generate", "-seed", "7", "-out", folder}, logs.Discard())
	require.ErrorContains(t, err, "add -force")
	edited, _ := os.ReadFile(filepath.Join(folder, "map.json"))
	assert.Equal(t, "hand edit", string(edited), "nothing was written")

	require.NoError(t, run([]string{"generate", "-seed", "7", "-out", folder, "-force"}, logs.Discard()))
	assert.NoError(t, run([]string{"check", folder}, logs.Discard()), "rewritten whole")
}

// ADR 0007 test 19
func TestCheckNamesTheProblem(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "world")
	require.NoError(t, run([]string{"generate", "-seed", "7", "-out", folder}, logs.Discard()))
	require.NoError(t, os.Remove(filepath.Join(folder, "height.png")))
	assert.ErrorContains(t, run([]string{"check", folder}, logs.Discard()), "height.png")
	assert.ErrorIs(t, run([]string{"check"}, logs.Discard()), errUsage)
	assert.ErrorIs(t, run([]string{"draw"}, logs.Discard()), errUsage)
}

// ADR 0012 test 7
func TestCityRefusesToOverwriteWithoutForce(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "doma-1")
	require.NoError(t, run([]string{"city", "-out", folder}, logs.Discard()), "a new folder")
	require.NoError(t, os.WriteFile(filepath.Join(folder, "map.json"), []byte("hand edit"), 0o644))

	require.ErrorContains(t, run([]string{"city", "-out", folder}, logs.Discard()), "add -force")
	require.NoError(t, run([]string{"city", "-out", folder, "-force"}, logs.Discard()))
	assert.NoError(t, run([]string{"check", folder}, logs.Discard()), "rewritten whole")
}

// ADR 0012 test 8
func TestCheckNamesADomeLeadingToNoMapFolder(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "world")
	require.NoError(t, run([]string{"generate", "-seed", "7", "-out", folder}, logs.Discard()))
	assert.ErrorContains(t, run([]string{"check", folder}, logs.Discard()), `leads to "doma-1", which has no map folder`)
}
