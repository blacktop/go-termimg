package cmd

import (
	"testing"

	termimg "github.com/blacktop/go-termimg"
)

func TestValidatePlacementCoordinates(t *testing.T) {
	tests := []struct {
		name    string
		x       int
		y       int
		wantErr bool
	}{
		{name: "origin", x: 0, y: 0, wantErr: false},
		{name: "positive", x: 5, y: 7, wantErr: false},
		{name: "negative x", x: -1, y: 0, wantErr: true},
		{name: "negative y", x: 0, y: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlacementCoordinates(tt.x, tt.y)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestParsePlacementImageID(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantID  uint32
		wantSet bool
		wantErr bool
	}{
		{name: "empty", in: "", wantID: 0, wantSet: false, wantErr: false},
		{name: "valid", in: "42", wantID: 42, wantSet: true, wantErr: false},
		{name: "zero", in: "0", wantID: 0, wantSet: false, wantErr: true},
		{name: "non numeric", in: "abc", wantID: 0, wantSet: false, wantErr: true},
		{name: "overflow", in: "4294967296", wantID: 0, wantSet: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotSet, err := parsePlacementImageID(tt.in)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if gotID != tt.wantID {
				t.Fatalf("id mismatch: got %d want %d", gotID, tt.wantID)
			}
			if gotSet != tt.wantSet {
				t.Fatalf("set mismatch: got %t want %t", gotSet, tt.wantSet)
			}
		})
	}
}

func TestStripUnicodePlaceholderPayload(t *testing.T) {
	transfer := "\x1b_Ga=T,f=32,i=1,q=2,m=0;AAAA\x1b\\"
	rendered := transfer + "\x1b[38;2;1;2;3m" + termimg.PLACEHOLDER_CHAR + "abc\x1b[39m"

	got := stripUnicodePlaceholderPayload(rendered)
	if got != transfer {
		t.Fatalf("unexpected strip result: got %q want %q", got, transfer)
	}
}

func TestStripUnicodePlaceholderPayloadNoPlaceholder(t *testing.T) {
	in := "\x1b_Ga=T,f=32,i=1,q=2,m=0;AAAA\x1b\\"
	got := stripUnicodePlaceholderPayload(in)
	if got != in {
		t.Fatalf("expected unchanged string, got %q", got)
	}
}
