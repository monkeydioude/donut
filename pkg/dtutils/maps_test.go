package dtutils

import (
	"slices"
	"testing"
)

func TestICanMapMatchKeysAllFunc(t *testing.T) {
	test := map[string]int{"A": 1, "B": 2, "C": 1}
	if !MapMatchKeysAll(test, 1) {
		t.Fail()
	}
	if !MapMatchKeysAll(test, 1, "A", "C") {
		t.Fail()
	}
	if MapMatchKeysAll(test, 1, "B") {
		t.Fail()
	}
	if !MapMatchKeysAll(test, 2, "B") {
		t.Fail()
	}
}

func TestICanMGetKeysByValue(t *testing.T) {
	test := map[string]int{"A": 1, "B": 2, "C": 1}
	if slices.Compare(MapKeysByValue(test, 1), []string{"A", "C"}) != 0 {
		t.Fail()
	}
}
