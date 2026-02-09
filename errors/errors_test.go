package errors

import (
	stderrors "errors"
	"io"
	"testing"
)

func TestIsEOF(t *testing.T) {
	if IsEOF(nil) {
		t.Fatalf("IsEOF(nil) = true, want false")
	}
	if !IsEOF(ErrEOF) {
		t.Fatalf("IsEOF(ErrEOF) = false, want true")
	}
	if !IsEOF(io.EOF) {
		t.Fatalf("IsEOF(io.EOF) = false, want true")
	}
	if IsEOF(stderrors.New("other")) {
		t.Fatalf("IsEOF(other) = true, want false")
	}
}
