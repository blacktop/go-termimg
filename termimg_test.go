package termimg

import (
	_ "image/jpeg"
	_ "image/png"
	"testing"
)

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		name string
		want Protocol
	}{
		// {
		// 	name: "iTerm2",
		// 	want: ITerm2,
		// },
		// {
		// 	name: "Kitty",
		// 	want: Kitty,
		// },
		{
			name: "Unsupported",
			want: Unsupported,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectProtocol(); got != tt.want {
				t.Errorf("DetectProtocol() = %v, want %s", got, tt.want)
			}
		})
	}
}
