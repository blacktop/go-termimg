package termimg

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// TerminalCapabilities represents the detected capabilities of the terminal
type TerminalCapabilities struct {
	// Graphics Protocol Support
	KittyGraphics    bool
	SixelGraphics    bool
	ITerm2Graphics   bool
	
	// Font and Size Information
	FontWidth        int
	FontHeight       int
	WindowCols       int
	WindowRows       int
	WindowPixelWidth int
	WindowPixelHeight int
	
	// Feature Support
	RectangularOps   bool        // Sixel rectangular editing operations
	MouseSupport     bool        // Basic mouse support detection
	TrueColor        bool        // 24-bit color support
	DeviceAttribs    []string    // Raw device attributes
	
	// Environment Information  
	IsTmux           bool
	IsScreen         bool
	TermName         string
	TermProgram      string
}

// CSIQuery represents a Control Sequence Introducer query
type CSIQuery struct {
	Query       string        // The query string to send
	Timeout     time.Duration // How long to wait for response
	Description string        // Human readable description
}

// CSIResponse represents a parsed response from a CSI query
type CSIResponse struct {
	Raw      string            // Raw response received
	Type     string            // Type of response (DA1, DA2, DECRPM, etc.)
	Values   []int             // Parsed numeric values
	Flags    map[string]bool   // Boolean flags extracted
	Metadata map[string]string // Additional metadata
}

// Standard CSI queries based on xterm documentation and widespread support
var (
	// Device Attributes queries - widely supported
	QueryDeviceAttribs1 = CSIQuery{
		Query:       "\x1b[c",
		Timeout:     200 * time.Millisecond,
		Description: "Primary Device Attributes (DA1)",
	}
	
	QueryDeviceAttribs2 = CSIQuery{
		Query:       "\x1b[>c", 
		Timeout:     200 * time.Millisecond,
		Description: "Secondary Device Attributes (DA2)",
	}
	
	// Font/cell size queries - reliable on most terminals
	CSIQueryFontSize = CSIQuery{
		Query:       "\x1b[16t",
		Timeout:     300 * time.Millisecond,
		Description: "Character cell size in pixels",
	}
	
	QueryWindowSizePixels = CSIQuery{
		Query:       "\x1b[14t",
		Timeout:     200 * time.Millisecond,
		Description: "Window size in pixels",
	}
	
	QueryWindowSizeChars = CSIQuery{
		Query:       "\x1b[18t",
		Timeout:     200 * time.Millisecond,
		Description: "Window size in characters",
	}
	
	// Cursor position - universally supported
	QueryCursorPosition = CSIQuery{
		Query:       "\x1b[6n",
		Timeout:     200 * time.Millisecond,
		Description: "Cursor Position Report (CPR)",
	}
	
	// Device Status Report - universally supported, good for basic connectivity test
	QueryDeviceStatus = CSIQuery{
		Query:       "\x1b[5n",
		Timeout:     200 * time.Millisecond,
		Description: "Device Status Report",
	}
	
	// Kitty graphics query - specific to Kitty protocol
	QueryKittyGraphics = CSIQuery{
		Query:       "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\",
		Timeout:     200 * time.Millisecond,
		Description: "Kitty Graphics Protocol Query",
	}
	
	// iTerm2 proprietary query - less reliable
	QueryITerm2 = CSIQuery{
		Query:       "\x1b[1337n",
		Timeout:     200 * time.Millisecond,
		Description: "iTerm2 Proprietary Query",
	}
)

// DetectTerminalCapabilities performs comprehensive terminal capability detection
func DetectTerminalCapabilities() (*TerminalCapabilities, error) {
	caps := &TerminalCapabilities{
		TermName:    os.Getenv("TERM"),
		TermProgram: os.Getenv("TERM_PROGRAM"),
		IsTmux:      inTmux(),
		IsScreen:    inScreen(),
		DeviceAttribs: make([]string, 0),
	}
	
	// Fast path: check environment variables first
	detectFromEnvironment(caps)
	
	// If not in an interactive terminal, return environment-based detection
	if !isInteractiveTerminal() {
		return caps, nil
	}
	
	// Perform CSI queries for detailed capability detection
	if err := detectFromCSIQueries(caps); err != nil {
		// Continue with environment-based detection if CSI queries fail
		return caps, nil
	}
	
	return caps, nil
}

// detectFromEnvironment performs fast capability detection using environment variables
func detectFromEnvironment(caps *TerminalCapabilities) {
	termName := strings.ToLower(caps.TermName)
	termProgram := caps.TermProgram
	
	// Kitty graphics detection
	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "":
		caps.KittyGraphics = true
	case strings.Contains(termName, "kitty"):
		caps.KittyGraphics = true
	case termProgram == "ghostty":
		caps.KittyGraphics = true
	case termProgram == "WezTerm":
		caps.KittyGraphics = true
		caps.ITerm2Graphics = true // WezTerm supports both
	case termProgram == "rio":
		caps.KittyGraphics = true
		caps.ITerm2Graphics = true
	}
	
	// Sixel graphics detection
	switch {
	case strings.Contains(termName, "sixel"):
		caps.SixelGraphics = true
	case strings.Contains(termName, "mlterm"):
		caps.SixelGraphics = true
	case strings.Contains(termName, "foot"):
		caps.SixelGraphics = true
	case strings.Contains(termName, "wezterm"):
		caps.SixelGraphics = true
	case strings.Contains(termName, "alacritty"):
		caps.SixelGraphics = true
	case termProgram == "iTerm.app":
		caps.SixelGraphics = true
		caps.ITerm2Graphics = true
	case termProgram == "mintty":
		caps.SixelGraphics = true
		caps.ITerm2Graphics = true
	}
	
	// iTerm2 graphics detection
	switch {
	case termProgram == "iTerm.app":
		caps.ITerm2Graphics = true
	case termProgram == "vscode" && os.Getenv("TERM_PROGRAM_VERSION") != "":
		caps.ITerm2Graphics = true
	case termProgram == "mintty":
		caps.ITerm2Graphics = true
	case termProgram == "WarpTerminal":
		caps.ITerm2Graphics = true
	case strings.Contains(strings.ToLower(os.Getenv("LC_TERMINAL")), "iterm"):
		caps.ITerm2Graphics = true
	}
	
	// True color support detection (24-bit color)
	switch {
	case strings.Contains(termName, "truecolor"):
		caps.TrueColor = true
	case strings.Contains(termName, "24bit"):
		caps.TrueColor = true
	case termProgram == "iTerm.app":
		caps.TrueColor = true
	case termProgram == "WezTerm":
		caps.TrueColor = true
	case strings.Contains(termName, "kitty"):
		caps.TrueColor = true
	case os.Getenv("COLORTERM") == "truecolor":
		caps.TrueColor = true
	case os.Getenv("COLORTERM") == "24bit":
		caps.TrueColor = true
	}
	
	// Set font size fallbacks based on terminal type
	caps.FontWidth, caps.FontHeight = getFontSizeFallback()
}

// detectFromCSIQueries performs detailed capability detection using CSI queries
func detectFromCSIQueries(caps *TerminalCapabilities) error {
	// Save current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to enter raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	
	// Buffer for collecting all responses
	responseBuffer := make([]byte, 0, 1024)
	
	// Send all queries in sequence with proper timing
	queries := []CSIQuery{
		QueryDeviceStatus,      // First - ensures terminal is responsive
		QueryDeviceAttribs1,    // Primary device attributes (sixel detection)
		QueryDeviceAttribs2,    // Secondary device attributes (terminal ID)
		CSIQueryFontSize,       // Font size in pixels
		QueryWindowSizePixels,  // Window size in pixels
		QueryWindowSizeChars,   // Window size in characters
		QueryKittyGraphics,     // Kitty graphics support
		QueryITerm2,           // iTerm2 support (less reliable)
	}
	
	// Send all queries
	for _, query := range queries {
		wrappedQuery := wrapQueryForMultiplexer(query.Query, caps.IsTmux)
		if _, err := os.Stdout.WriteString(wrappedQuery); err != nil {
			continue // Skip failed queries
		}
		time.Sleep(10 * time.Millisecond) // Small delay between queries
	}
	
	// Collect responses with timeout
	responseChan := make(chan []byte, 1)
	go func() {
		buffer := make([]byte, 1024)
		deadline := time.Now().Add(500 * time.Millisecond)
		
		for time.Now().Before(deadline) {
			// Set a short read timeout
			if err := os.Stdin.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
				break
			}
			
			n, err := os.Stdin.Read(buffer[len(responseBuffer):])
			if err != nil {
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					continue // Continue reading on timeout
				}
				break
			}
			
			if n > 0 {
				responseBuffer = append(responseBuffer, buffer[len(responseBuffer):len(responseBuffer)+n]...)
				deadline = time.Now().Add(100 * time.Millisecond) // Extend deadline if receiving data
			}
		}
		
		responseChan <- responseBuffer
	}()
	
	// Wait for responses
	select {
	case responses := <-responseChan:
		parseCSIResponses(string(responses), caps)
	case <-time.After(600 * time.Millisecond):
		// Timeout - continue with environment-based detection
	}
	
	return nil
}

// parseCSIResponses parses the collected terminal responses and updates capabilities
func parseCSIResponses(responses string, caps *TerminalCapabilities) {
	// Split responses by escape sequences
	parts := strings.Split(responses, "\x1b")
	
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		
		response := "\x1b" + part
		parsed := parseCSIResponse(response)
		
		switch parsed.Type {
		case "DA1": // Primary Device Attributes
			caps.DeviceAttribs = append(caps.DeviceAttribs, parsed.Raw)
			// Check for Sixel support (capability 4)
			for _, val := range parsed.Values {
				if val == 4 {
					caps.SixelGraphics = true
				}
				if val == 28 {
					caps.RectangularOps = true
				}
			}
			
		case "DA2": // Secondary Device Attributes
			caps.DeviceAttribs = append(caps.DeviceAttribs, parsed.Raw)
			
		case "FONT_SIZE": // CSI 16 t response
			if len(parsed.Values) >= 2 {
				caps.FontHeight = parsed.Values[0]
				caps.FontWidth = parsed.Values[1]
			}
			
		case "WINDOW_SIZE_PIXELS": // CSI 14 t response
			if len(parsed.Values) >= 2 {
				caps.WindowPixelHeight = parsed.Values[0]
				caps.WindowPixelWidth = parsed.Values[1]
			}
			
		case "WINDOW_SIZE_CHARS": // CSI 18 t response
			if len(parsed.Values) >= 2 {
				caps.WindowRows = parsed.Values[0]
				caps.WindowCols = parsed.Values[1]
			}
			
		case "KITTY_OK": // Kitty graphics response
			caps.KittyGraphics = true
			
		case "ITERM2_OK": // iTerm2 response
			caps.ITerm2Graphics = true
			
		case "DSR": // Device Status Report
			// Terminal is responsive
		}
	}
}

// parseCSIResponse parses a single CSI response into structured data
func parseCSIResponse(response string) CSIResponse {
	parsed := CSIResponse{
		Raw:      response,
		Values:   make([]int, 0),
		Flags:    make(map[string]bool),
		Metadata: make(map[string]string),
	}
	
	if !strings.HasPrefix(response, "\x1b") {
		return parsed
	}
	
	// Remove escape prefix
	content := response[1:]
	
	switch {
	case strings.HasPrefix(content, "[?") && strings.HasSuffix(content, "c"):
		// Primary Device Attributes: \x1b[?1;2;4;6;9;15;18;21;22c
		parsed.Type = "DA1"
		inner := content[2 : len(content)-1] // Remove [? and c
		parts := strings.Split(inner, ";")
		for _, part := range parts {
			if val, err := strconv.Atoi(part); err == nil {
				parsed.Values = append(parsed.Values, val)
			}
		}
		
	case strings.HasPrefix(content, "[>") && strings.HasSuffix(content, "c"):
		// Secondary Device Attributes: \x1b[>1;95;0c
		parsed.Type = "DA2"
		inner := content[2 : len(content)-1] // Remove [> and c
		parts := strings.Split(inner, ";")
		for _, part := range parts {
			if val, err := strconv.Atoi(part); err == nil {
				parsed.Values = append(parsed.Values, val)
			}
		}
		
	case strings.HasPrefix(content, "[6;") && strings.HasSuffix(content, "t"):
		// Font size response: \x1b[6;height;width;t
		parsed.Type = "FONT_SIZE"
		inner := content[3 : len(content)-1] // Remove [6; and t
		parts := strings.Split(inner, ";")
		for _, part := range parts {
			if val, err := strconv.Atoi(part); err == nil {
				parsed.Values = append(parsed.Values, val)
			}
		}
		
	case strings.HasPrefix(content, "[4;") && strings.HasSuffix(content, "t"):
		// Window size in pixels: \x1b[4;height;width;t
		parsed.Type = "WINDOW_SIZE_PIXELS"
		inner := content[3 : len(content)-1] // Remove [4; and t
		parts := strings.Split(inner, ";")
		for _, part := range parts {
			if val, err := strconv.Atoi(part); err == nil {
				parsed.Values = append(parsed.Values, val)
			}
		}
		
	case strings.HasPrefix(content, "[8;") && strings.HasSuffix(content, "t"):
		// Window size in characters: \x1b[8;rows;cols;t
		parsed.Type = "WINDOW_SIZE_CHARS"
		inner := content[3 : len(content)-1] // Remove [8; and t
		parts := strings.Split(inner, ";")
		for _, part := range parts {
			if val, err := strconv.Atoi(part); err == nil {
				parsed.Values = append(parsed.Values, val)
			}
		}
		
	case strings.HasPrefix(content, "_Gi=31;OK"):
		// Kitty graphics response
		parsed.Type = "KITTY_OK"
		
	case strings.Contains(content, "1337"):
		// iTerm2 response
		parsed.Type = "ITERM2_OK"
		
	case strings.HasPrefix(content, "[0n"):
		// Device Status Report - terminal OK
		parsed.Type = "DSR"
		
	case strings.Contains(content, "R"):
		// Cursor Position Report
		parsed.Type = "CPR"
	}
	
	return parsed
}

// SendCSIQuery sends a single CSI query and returns the parsed response
func SendCSIQuery(query CSIQuery) (*CSIResponse, error) {
	if !isInteractiveTerminal() {
		return nil, fmt.Errorf("not an interactive terminal")
	}
	
	// Save current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("failed to enter raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	
	// Wrap query for terminal multiplexers
	wrappedQuery := wrapQueryForMultiplexer(query.Query, inTmux())
	
	// Send query
	if _, err := os.Stdout.WriteString(wrappedQuery); err != nil {
		return nil, fmt.Errorf("failed to send query: %w", err)
	}
	
	// Collect response
	responseChan := make(chan string, 1)
	go func() {
		buffer := make([]byte, 256)
		n, err := os.Stdin.Read(buffer)
		if err != nil || n == 0 {
			responseChan <- ""
			return
		}
		responseChan <- string(buffer[:n])
	}()
	
	// Wait for response with timeout
	select {
	case response := <-responseChan:
		parsed := parseCSIResponse(response)
		return &parsed, nil
	case <-time.After(query.Timeout):
		return nil, fmt.Errorf("query timeout after %v", query.Timeout)
	}
}

// wrapQueryForMultiplexer wraps CSI queries for terminal multiplexers like tmux/screen
func wrapQueryForMultiplexer(query string, isTmux bool) string {
	if isTmux {
		// Tmux passthrough: \ePtmux;\e<query>\e\\
		return "\x1bPtmux;\x1b" + strings.ReplaceAll(query, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	
	// Screen multiplexer could be handled here if needed:
	// if isScreen() {
	//     return "\x1bP" + query + "\x1b\\"
	// }
	
	return query
}

// QueryTerminalFontSize queries the terminal for actual font dimensions in pixels
func QueryTerminalFontSize() (width, height int, err error) {
	response, err := SendCSIQuery(CSIQueryFontSize)
	if err != nil {
		return 0, 0, err
	}
	
	if response.Type == "FONT_SIZE" && len(response.Values) >= 2 {
		return response.Values[1], response.Values[0], nil
	}
	
	return 0, 0, fmt.Errorf("invalid font size response")
}

// QueryWindowSize queries the terminal for window dimensions
func QueryWindowSize() (cols, rows, pixelWidth, pixelHeight int, err error) {
	// Try to get both character and pixel dimensions
	charResponse, err1 := SendCSIQuery(QueryWindowSizeChars)
	pixelResponse, err2 := SendCSIQuery(QueryWindowSizePixels)
	
	if err1 != nil && err2 != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to query window size")
	}
	
	if err1 == nil && charResponse.Type == "WINDOW_SIZE_CHARS" && len(charResponse.Values) >= 2 {
		rows = charResponse.Values[0]
		cols = charResponse.Values[1]
	}
	
	if err2 == nil && pixelResponse.Type == "WINDOW_SIZE_PIXELS" && len(pixelResponse.Values) >= 2 {
		pixelHeight = pixelResponse.Values[0]
		pixelWidth = pixelResponse.Values[1]
	}
	
	return cols, rows, pixelWidth, pixelHeight, nil
}

// QueryDeviceAttributes queries terminal for device attributes (capabilities)
func QueryDeviceAttributes() (primary, secondary []int, err error) {
	// Query primary device attributes
	response1, err1 := SendCSIQuery(QueryDeviceAttribs1)
	if err1 == nil && response1.Type == "DA1" {
		primary = response1.Values
	}
	
	// Query secondary device attributes  
	response2, err2 := SendCSIQuery(QueryDeviceAttribs2)
	if err2 == nil && response2.Type == "DA2" {
		secondary = response2.Values
	}
	
	if err1 != nil && err2 != nil {
		return nil, nil, fmt.Errorf("failed to query device attributes")
	}
	
	return primary, secondary, nil
}

// Helper functions for environment detection - these reference existing functions

// inTmux is already defined in renderers.go
// inScreen checks if running inside GNU Screen
func inScreen() bool {
	return strings.HasPrefix(os.Getenv("TERM"), "screen")
}

// isInteractiveTerminal is already defined in renderers.go
// getFontSizeFallback is already defined in renderers.go
// getTerminalFontSize is already defined in renderers.go

// Global capability cache to avoid repeated queries
var (
	globalCapabilities     *TerminalCapabilities
	capabilitiesCached     bool
)

// GetCachedCapabilities returns cached terminal capabilities, detecting them if needed
func GetCachedCapabilities() *TerminalCapabilities {
	if !capabilitiesCached {
		if caps, err := DetectTerminalCapabilities(); err == nil {
			globalCapabilities = caps
		} else {
			// Create minimal capabilities from environment if detection fails
			globalCapabilities = &TerminalCapabilities{
				TermName:    os.Getenv("TERM"),
				TermProgram: os.Getenv("TERM_PROGRAM"),
				IsTmux:      inTmux(),
				IsScreen:    inScreen(),
			}
			detectFromEnvironment(globalCapabilities)
		}
		capabilitiesCached = true
	}
	
	return globalCapabilities
}

// RefreshCapabilities forces a refresh of the cached terminal capabilities
func RefreshCapabilities() (*TerminalCapabilities, error) {
	capabilitiesCached = false
	globalCapabilities = nil
	return DetectTerminalCapabilities()
}
