package coords

import "testing"

func TestICanComputeIsometricAndBackToCartesian(t *testing.T) {
	tw, th := 100, 50
	iso := Coords{10, 5}.IntoIsometric(tw, th)
	if iso.X != 250 || iso.Y != 375 {
		t.Fail()
	}
	cart := iso.IntoCartesian()
	if cart.X != 10 || cart.Y != 5 {
		t.Fail()
	}
}
