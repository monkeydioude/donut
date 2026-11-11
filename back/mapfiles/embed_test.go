package mapfiles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"donut/internal/worldmap/domain"
	"donut/mapfiles"
)

// ADR 0007 test 17
func TestTheCommittedWorldPassesEveryCheck(t *testing.T) {
	world, err := mapfiles.Read("world")
	require.NoError(t, err)
	assert.Equal(t, domain.Summary{Slug: "world", Name: "World", Width: 2048, Height: 1024, Wraps: true}, world.Summary)
	assert.Len(t, world.Regions, 18)
	assert.NotEmpty(t, world.Objects)
}

func TestReadRefusesAMapItDoesNotHave(t *testing.T) {
	for _, slug := range []string{"nowhere", "../world", "world/..", ""} {
		_, err := mapfiles.Read(slug)
		assert.Error(t, err, "%q", slug)
	}
}
