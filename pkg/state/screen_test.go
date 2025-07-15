package state

import "testing"

func TestICanNormalize(t *testing.T) {
	screen := NewScreen(100, 50)
	tx, ty := screen.Normalize(100, 50, 200, 100)
	if tx != 50 || ty != 25 {
		t.Fail()
	}
}
