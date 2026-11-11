package domain

import (
	"sort"

	core "donut/internal/core/domain"
)

// levels is how many heights a cell may have, from MinHeight to MaxHeight.
const levels = MaxHeight - MinHeight + 1

// ReliefOptions shape the hills and valleys (ADR 0007). No wavelengths: flat ground.
type ReliefOptions struct {
	// Wavelengths are the noise's octaves, in cells, the first one the strongest, each next one half as strong.
	// On a wrapping map, each should divide the map's sides, so the noise has no seam.
	Wavelengths []int
	// Shares are the share of cells at each level, from MinHeight to MaxHeight, before cliffs are smoothed.
	Shares [levels]float64
}

// starterRelief: about half the cells at 0, a quarter above, a quarter below, and 2% at each end.
func starterRelief() ReliefOptions {
	return ReliefOptions{Wavelengths: []int{64, 32, 16}, Shares: [levels]float64{0.02, 0.07, 0.16, 0.50, 0.16, 0.07, 0.02}}
}

// raise gives each land cell a level from smooth noise plus what it gains inland, cut by the shares over the land
// cells, then smooths the cliffs away. The ocean stays where it is.
func (l *land) raise(o ReliefOptions, rise func(i int) float64, rng Rng) {
	if len(o.Wavelengths) == 0 {
		return
	}
	noise := fractalNoise(l.grid, o.Wavelengths, rng)
	var onLand []float64
	for i := range noise {
		if l.landmass[i] >= 0 {
			noise[i] += rise(i)
			onLand = append(onLand, noise[i])
		}
	}
	thresholds := levelThresholds(onLand, o.Shares)
	for i, value := range noise {
		if l.landmass[i] >= 0 {
			l.height[i] = levelOf(value, thresholds)
		}
	}
	l.smoothCliffs()
}

// fractalNoise adds up octaves of value noise, each half as strong as the one before, scaled to −1…1. All zero
// without wavelengths.
func fractalNoise(grid core.Grid, wavelengths []int, rng Rng) []float64 {
	noise := make([]float64, grid.Width*grid.Height)
	strength, total := 1.0, 0.0
	for _, wavelength := range wavelengths {
		octave := newValueNoise(grid, wavelength, rng)
		for i := range noise {
			noise[i] += strength * octave.at(i%grid.Width, i/grid.Width)
		}
		total += strength
		strength /= 2
	}
	for i := range noise {
		if total > 0 {
			noise[i] /= total
		}
	}
	return noise
}

// valueNoise is random values on a lattice, one every `wavelength` cells, smoothly blended in between. The lattice
// wraps, so the noise has no seam when the wavelength divides the map's sides.
type valueNoise struct {
	wavelength int
	cols, rows int
	values     []float64
}

// newValueNoise draws the lattice's values in [−1, 1), row by row.
func newValueNoise(grid core.Grid, wavelength int, rng Rng) valueNoise {
	cols, rows := ceilDiv(grid.Width, wavelength), ceilDiv(grid.Height, wavelength)
	values := make([]float64, cols*rows)
	for i := range values {
		values[i] = rng.Float64()*2 - 1
	}
	return valueNoise{wavelength: wavelength, cols: cols, rows: rows, values: values}
}

func (n valueNoise) at(col, row int) float64 {
	x0, y0 := col/n.wavelength, row/n.wavelength
	tx := smoothstep(float64(col%n.wavelength) / float64(n.wavelength))
	ty := smoothstep(float64(row%n.wavelength) / float64(n.wavelength))
	top := lerp(n.value(x0, y0), n.value(x0+1, y0), tx)
	bottom := lerp(n.value(x0, y0+1), n.value(x0+1, y0+1), tx)
	return lerp(top, bottom, ty)
}

func (n valueNoise) value(x, y int) float64 {
	return n.values[(y%n.rows)*n.cols+x%n.cols]
}

// levelThresholds are the noise values between levels: the shares' running totals, as quantiles of the noise.
func levelThresholds(noise []float64, shares [levels]float64) [levels - 1]float64 {
	sorted := append([]float64(nil), noise...)
	sort.Float64s(sorted)
	total := 0.0
	for _, share := range shares {
		total += share
	}
	var thresholds [levels - 1]float64
	runningTotal := 0.0
	for i := range thresholds {
		runningTotal += shares[i] / total
		thresholds[i] = sorted[min(len(sorted)-1, int(runningTotal*float64(len(sorted))))]
	}
	return thresholds
}

// levelOf is the level of a noise value: MinHeight under the first threshold, one more past each.
func levelOf(value float64, thresholds [levels - 1]float64) int8 {
	level := int8(MinHeight)
	for _, threshold := range thresholds {
		if value >= threshold {
			level++
		}
	}
	return level
}

func smoothstep(t float64) float64 {
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func ceilDiv(a, b int) int {
	return (a + b - 1) / b
}
