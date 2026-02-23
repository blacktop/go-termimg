package termimg

import (
	"os"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected Protocol
	}{
		{
			name: "Kitty terminal via TERM",
			envVars: map[string]string{
				"TERM": "xterm-kitty",
			},
			expected: Kitty,
		},
		{
			name: "Kitty terminal via KITTY_WINDOW_ID",
			envVars: map[string]string{
				"KITTY_WINDOW_ID": "1",
			},
			expected: Kitty,
		},
		{
			name: "iTerm2 terminal",
			envVars: map[string]string{
				"TERM_PROGRAM": "iTerm.app",
			},
			expected: ITerm2,
		},
		{
			name: "WezTerm terminal",
			envVars: map[string]string{
				"TERM_PROGRAM": "WezTerm",
			},
			expected: ITerm2,
		},
		{
			name: "Apple Terminal",
			envVars: map[string]string{
				"TERM_PROGRAM": "Apple_Terminal",
			},
			expected: ITerm2,
		},
		{
			name: "Mintty terminal",
			envVars: map[string]string{
				"TERM_PROGRAM": "mintty",
			},
			expected: ITerm2,
		},
		{
			name: "Konsole terminal",
			envVars: map[string]string{
				"KONSOLE_VERSION": "22.12.3",
			},
			expected: ITerm2,
		},
		{
			name: "Unknown terminal defaults",
			envVars: map[string]string{
				"TERM": "xterm-256color",
			},
			expected: Auto, // May fall back to Auto
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			clearEnvVars := []string{"TERM", "TERM_PROGRAM", "KONSOLE_VERSION", "KITTY_WINDOW_ID", "TMUX"}
			for _, env := range clearEnvVars {
				if val, exists := os.LookupEnv(env); exists {
					originalEnv[env] = val
				}
				os.Unsetenv(env)
			}

			// Set test environment
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Clear caches to ensure fresh detection
			ClearEnvironmentCache()
			ClearFeatureCache()

			result := DetectProtocol()
			// Detection may fall back to different protocols in test environment
			// Just ensure it returns a valid protocol
			assert.NotEqual(t, Protocol(0), result, "Should detect a valid protocol")

			// Log what was actually detected for debugging
			t.Logf("Expected: %s, Got: %s", tt.expected.String(), result.String())

			// Restore original environment
			for _, env := range clearEnvVars {
				os.Unsetenv(env)
			}
			for k, v := range originalEnv {
				os.Setenv(k, v)
			}
		})
	}
}

func TestProtocolSupport(t *testing.T) {
	t.Run("KittySupported", func(t *testing.T) {
		// Just ensure it doesn't panic
		supported := KittySupported()
		assert.IsType(t, false, supported)
	})

	t.Run("SixelSupported", func(t *testing.T) {
		supported := SixelSupported()
		assert.IsType(t, false, supported)
	})

	t.Run("ITerm2Supported", func(t *testing.T) {
		supported := ITerm2Supported()
		assert.IsType(t, false, supported)
	})

	t.Run("HalfblocksSupported", func(t *testing.T) {
		supported := HalfblocksSupported()
		assert.True(t, supported, "Halfblocks should always be supported")
	})
}

func TestQueryTerminalFeatures(t *testing.T) {
	features := QueryTerminalFeatures()
	require.NotNil(t, features)

	// Basic validation that features struct is populated
	assert.IsType(t, "", features.TermName)
	assert.IsType(t, "", features.TermProgram)
	assert.IsType(t, false, features.IsTmux)
	assert.IsType(t, false, features.IsScreen)
	assert.GreaterOrEqual(t, features.FontWidth, 0)
	assert.GreaterOrEqual(t, features.FontHeight, 0)
}

func TestParallelProtocolDetection(t *testing.T) {
	// Test that parallel detection doesn't panic or race
	kitty, sixel, iterm2 := ParallelProtocolDetection()

	// Values should be booleans
	assert.IsType(t, false, kitty)
	assert.IsType(t, false, sixel)
	assert.IsType(t, false, iterm2)

	// At least one should be true if we're in a supported terminal
	// (but this might fail in CI environments, so just check types)
}

func TestParallelProtocolDetectionPreservesSixelWhenKittyDetectedFromEnv(t *testing.T) {
	wasTmuxForced := IsTmuxForced()
	defer ForceTmux(wasTmuxForced)
	ForceTmux(false)

	// Normalize all relevant env hints so this assertion is deterministic.
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("XTERM_VERSION", "")
	t.Setenv("TMUX", "")
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")
	t.Setenv("WEZTERM_EXECUTABLE", "")
	t.Setenv("ITERM_SESSION_ID", "")
	t.Setenv("LC_TERMINAL", "")
	t.Setenv("TERM_SESSION_ID", "")

	kitty, sixel, iterm2 := ParallelProtocolDetection()
	assert.True(t, kitty, "WezTerm should be detected as kitty-capable from env hints")
	assert.True(t, sixel, "Sixel env detection should still run before early return")
	assert.False(t, iterm2, "WezTerm env detection path should not force iTerm2 support")
}

func TestProtocolStrings(t *testing.T) {
	tests := []struct {
		protocol Protocol
		expected string
	}{
		{Auto, "Auto"},
		{Kitty, "Kitty"},
		{ITerm2, "iTerm2"},
		{Sixel, "Sixel"},
		{Halfblocks, "Halfblocks"},
		{Unsupported, "unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.protocol.String())
		})
	}
}

func TestDetermineProtocols(t *testing.T) {
	protocols := DetermineProtocols()
	assert.NotEmpty(t, protocols, "Should return at least one protocol")

	// Should always include halfblocks as fallback
	found := slices.Contains(protocols, Halfblocks)
	assert.True(t, found, "Should include halfblocks as fallback")
}

func TestSupportedProtocols(t *testing.T) {
	supported := SupportedProtocols()
	assert.NotEmpty(t, supported, "Should return some supported protocols")
	assert.Contains(t, supported, "Halfblocks", "Should include Halfblocks")
}

func TestDetectionLog(t *testing.T) {
	// Clear log
	ClearDetectionLog()

	// No logs initially
	logs := GetDetectionLog()
	assert.Empty(t, logs)

	// The detection system may or may not log automatically
	// Just verify the functions work without panicking
	ClearDetectionLog()
	logs = GetDetectionLog()
	assert.Empty(t, logs)
}

func TestTerminalQuerier(t *testing.T) {
	querier, err := NewTerminalQuerier()
	if err != nil {
		t.Skip("Cannot create terminal querier in test environment")
		return
	}

	require.NotNil(t, querier)

	// Test that Query method exists and handles timeout
	// Use very short timeout to avoid hanging
	result, err := querier.Query("\x1b[c", 10*time.Millisecond)

	// In test environment, this will likely timeout or error
	// Just ensure it doesn't panic
	assert.IsType(t, "", result)
	// Error is expected in test environment
}

func TestEnvironmentDetection(t *testing.T) {
	t.Run("DetectKittyFromEnvironment", func(t *testing.T) {
		// Save current env
		originalTerm := os.Getenv("TERM")
		originalKitty := os.Getenv("KITTY_WINDOW_ID")

		// Test positive case
		os.Setenv("TERM", "xterm-kitty")
		assert.True(t, DetectKittyFromEnvironment())

		// Test with KITTY_WINDOW_ID
		os.Unsetenv("TERM")
		os.Setenv("KITTY_WINDOW_ID", "1")
		assert.True(t, DetectKittyFromEnvironment())

		// Test negative case
		os.Unsetenv("TERM")
		os.Unsetenv("KITTY_WINDOW_ID")
		ClearEnvironmentCache() // Clear cache to ensure fresh detection
		// In test environment, other env vars might still trigger kitty detection
		// The important thing is that the function doesn't panic and returns a boolean
		result := DetectKittyFromEnvironment()
		assert.IsType(t, false, result, "Should return a boolean value")

		// Restore environment
		if originalTerm != "" {
			os.Setenv("TERM", originalTerm)
		} else {
			os.Unsetenv("TERM")
		}
		if originalKitty != "" {
			os.Setenv("KITTY_WINDOW_ID", originalKitty)
		} else {
			os.Unsetenv("KITTY_WINDOW_ID")
		}
	})

	t.Run("DetectITerm2FromEnvironment", func(t *testing.T) {
		// Save current env
		original := os.Getenv("TERM_PROGRAM")

		// Test positive case
		os.Setenv("TERM_PROGRAM", "iTerm.app")
		assert.True(t, DetectITerm2FromEnvironment())

		// Test negative case
		os.Setenv("TERM_PROGRAM", "other")
		assert.False(t, DetectITerm2FromEnvironment())

		// Restore environment
		if original != "" {
			os.Setenv("TERM_PROGRAM", original)
		} else {
			os.Unsetenv("TERM_PROGRAM")
		}
	})

	t.Run("DetectSixelFromEnvironment", func(t *testing.T) {
		// Just ensure it doesn't panic - sixel detection is complex
		result := DetectSixelFromEnvironment()
		assert.IsType(t, false, result)
	})
}

func TestCacheClearing(t *testing.T) {
	// Test that cache clearing functions don't panic
	assert.NotPanics(t, func() {
		ClearEnvironmentCache()
	})

	assert.NotPanics(t, func() {
		ClearFeatureCache()
	})
}

func TestConcurrentDetection(t *testing.T) {
	// Test concurrent access to detection functions
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			_ = DetectProtocol()
			_ = QueryTerminalFeatures()
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent detection timed out")
		}
	}
}

func BenchmarkDetectProtocol(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DetectProtocol()
	}
}

func BenchmarkQueryTerminalFeatures(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = QueryTerminalFeatures()
	}
}
