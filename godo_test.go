package godo

import (
	"testing"
)

func TestFormatCheckMarkTrue(t *testing.T) {
	want := "[X]"
	done := formatCheckMark(true)

	if want != done {
		t.Fatalf(`formatCheckMark(true) = %q, want match for %#q, nil`, done, want)
	}
}

func TestFormatCheckMarkFalse(t *testing.T) {
	want := "[ ]"
	done := formatCheckMark(false)

	if want != done {
		t.Fatalf(`formatCheckMark(true) = %q, want match for %#q, nil`, done, want)
	}
}
