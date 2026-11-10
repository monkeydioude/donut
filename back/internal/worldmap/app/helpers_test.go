package app_test

import "math/rand/v2"

// seeded is the rng EnsureStarterMap uses for a seed.
func seeded(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed))
}
