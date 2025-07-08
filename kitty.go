package termimg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"io"
	"os"
	"strings"
	"sync/atomic"

	"golang.org/x/term"
)

// reference: https://github.com/kovidgoyal/kitty/tree/master/kittens/icat

const (
	DATA_RGBA_32_BIT = "f=32" // default
	DATA_RGBA_24_BIT = "f=24"
	DATA_PNG         = "f=100"

	ACTION_TRANSFER  = "a=T"
	ACTION_DELETE    = "a=d"
	ACTION_QUERY     = "a=q"
	ACTION_ANIMATE   = "a=a"
	ACTION_PLACEMENT = "a=p"

	COMPRESS_ZLIB = "0=z"

	TRANSFER_DIRECT = "t=d"
	TRANSFER_FILE   = "t=f"
	TRANSFER_TEMP   = "t=t"
	TRANSFER_SHARED = "t=s"

	// More flags for continuation
	MORE_CHUNKS = "m=1" // More chunks follow
	FINAL_CHUNK = "m=0" // Final chunk (default)

	// Virtual placement
	VIRTUAL_PLACEMENT = "U=1" // Create virtual placement

	DELETE_WITH_ID          = "d=i"
	DELETE_NEWEST           = "d=n"
	DELETE_AT_CURSOR        = "d=c"
	DELETE_ANIMATION_FRAMES = "d=a"

	SUPPRESS_OK  = "q=1"
	SUPPRESS_ERR = "q=2"
)

// AnimationOptions contains parameters for Kitty image animation
type AnimationOptions struct {
	DelayMs  int      // Delay between frames in milliseconds
	Loops    int      // Number of animation loops (0 = infinite)
	ImageIDs []string // List of image IDs to animate between
}

// PositionOptions contains parameters for precise image positioning
type PositionOptions struct {
	X      int // X coordinate in character cells
	Y      int // Y coordinate in character cells
	ZIndex int // Z-index for layering
}

// KittyOptions contains Kitty-specific rendering options
type KittyOptions struct {
	ImageID      string
	Placement    string
	UseUnicode   bool
	Animation    *AnimationOptions
	Position     *PositionOptions
	FileTransfer bool
}

// Global image ID counter for Kitty protocol to ensure unique IDs across all renderer instances
var globalKittyImageID uint32

// KittyRenderer implements the Renderer interface for Kitty graphics protocol
type KittyRenderer struct {
	imageID   uint32
	lastID    uint32
	chunkSize int
}

// Protocol returns the protocol type
func (r *KittyRenderer) Protocol() Protocol {
	return Kitty
}

// Render generates the escape sequence for displaying the image
func (r *KittyRenderer) Render(img image.Image, opts RenderOptions) (string, error) {
	// Process the image (resize, dither, etc.)
	processed, err := processImage(img, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	// Initialize renderer if needed
	if r.chunkSize == 0 {
		r.chunkSize = BASE64_CHUNK_SIZE
	}

	// Generate unique image ID using atomic counter to ensure uniqueness across all instances
	imageID := atomic.AddUint32(&globalKittyImageID, 1)
	r.lastID = imageID

	// Get raw RGBA data
	bounds := processed.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, rgbaImg.Bounds(), processed, bounds.Min, draw.Src)
	data := rgbaImg.Pix

	// Build the escape sequence
	var output strings.Builder

	// Calculate dimensions
	pixelWidth := bounds.Dx()
	pixelHeight := bounds.Dy()

	// Get terminal font size for character cell calculations
	fontW, fontH := getTerminalFontSize()
	cols := pixelWidth / fontW
	rows := pixelHeight / fontH

	// If dimensions were specified in options, use those
	if opts.Width > 0 {
		cols = opts.Width
	}
	if opts.Height > 0 {
		rows = opts.Height
	}

	// If no dimensions specified, auto-detect terminal size
	if opts.Width == 0 && opts.Height == 0 && opts.ScaleMode == ScaleFit {
		if termWidth, termHeight, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			// For ScaleFit mode, calculate the scaling to fit within terminal
			srcW := float64(pixelWidth)
			srcH := float64(pixelHeight)

			// Convert terminal cells to approximate pixels
			termPixelW := float64(termWidth * fontW)
			termPixelH := float64(termHeight * fontH)

			// Calculate scaling ratios
			ratioW := termPixelW / srcW
			ratioH := termPixelH / srcH

			// Use the smaller ratio to fit within bounds
			ratio := min(ratioW, ratioH)

			// Calculate final dimensions in cells
			cols = int((srcW * ratio) / float64(fontW))
			rows = int((srcH * ratio) / float64(fontH))
		}
	}

	// Build control data (with quiet mode to suppress terminal responses)
	control := fmt.Sprintf("a=T,f=32,i=%d,s=%d,v=%d,c=%d,r=%d,q=2",
		imageID, pixelWidth, pixelHeight, cols, rows)

	// Add z-index if specified
	if opts.ZIndex > 0 {
		control += fmt.Sprintf(",z=%d", opts.ZIndex)
	}

	// Add virtual placement if specified
	if opts.Virtual {
		control += ",U=1"
	}

	// Send the image data in chunks
	first := true
	for i := 0; i < len(data); i += r.chunkSize {
		end := min(i+r.chunkSize, len(data))
		chunk := data[i:end]
		encodedChunk := base64.StdEncoding.EncodeToString(chunk)

		var chunkSequence string
		if first {
			first = false
			if len(data) > r.chunkSize {
				// More chunks to come
				chunkSequence = fmt.Sprintf("\x1b_G%s,m=1;%s\x1b\\", control, encodedChunk)
			} else {
				// Single chunk
				chunkSequence = fmt.Sprintf("\x1b_G%s;%s\x1b\\", control, encodedChunk)
			}
		} else {
			if end < len(data) {
				// Continuation chunk
				chunkSequence = fmt.Sprintf("\x1b_Gm=1,q=2;%s\x1b\\", encodedChunk)
			} else {
				// Last chunk
				chunkSequence = fmt.Sprintf("\x1b_Gm=0,q=2;%s\x1b\\", encodedChunk)
			}
		}

		output.WriteString(wrapTmuxPassthrough(chunkSequence))
	}

	// Handle Kitty-specific options
	if opts.KittyOpts != nil {
		// If virtual placement with unicode, generate placeholders
		if opts.Virtual && opts.KittyOpts.UseUnicode {
			placeholders := r.generateUnicodePlaceholders(imageID, cols, rows)
			output.WriteString(placeholders)
		}

		// Handle animation after image transfer
		if opts.KittyOpts.Animation != nil && len(opts.KittyOpts.Animation.ImageIDs) > 0 {
			// TODO: Animation is handled separately after all images are transferred
			// This is just to validate the option structure
		}

		// Handle positioning after image transfer
		if opts.KittyOpts.Position != nil {
			// TODO: Positioning is handled separately via PlaceImage method
			// This is just to validate the option structure
		}
	}

	return output.String(), nil
}

// Print outputs the image directly to stdout
func (r *KittyRenderer) Print(img image.Image, opts RenderOptions) error {
	// Check if we should use file transfer optimization
	if opts.KittyOpts != nil && opts.KittyOpts.FileTransfer {
		// TODO: File transfer would require knowing the source file path
		// This is best handled at a higher level in the Image API
		// For now, fall back to regular rendering
	}

	output, err := r.Render(img, opts)
	if err != nil {
		return err
	}
	_, err = io.WriteString(os.Stdout, output)

	// Handle post-render operations
	if err == nil && opts.KittyOpts != nil {
		// Handle positioning if specified
		if opts.KittyOpts.Position != nil {
			imageID := fmt.Sprintf("%d", r.lastID)
			err = r.PlaceImage(imageID, opts.KittyOpts.Position.X,
				opts.KittyOpts.Position.Y, opts.KittyOpts.Position.ZIndex)
		}

		// Handle animation if specified
		if opts.KittyOpts.Animation != nil && len(opts.KittyOpts.Animation.ImageIDs) > 0 {
			err = r.AnimateImages(opts.KittyOpts.Animation.ImageIDs,
				opts.KittyOpts.Animation.DelayMs, opts.KittyOpts.Animation.Loops)
		}
	}

	return err
}

// Clear removes the image from the terminal
func (r *KittyRenderer) Clear(opts ClearOptions) error {
	var control string

	if opts.All {
		// Clear all images
		control = "a=d"
	} else if opts.ImageID != "" {
		// Clear specific image by ID
		control = fmt.Sprintf("a=d,d=i,i=%s", opts.ImageID)
	} else if r.lastID > 0 {
		// Clear the last rendered image
		control = fmt.Sprintf("a=d,d=i,i=%d", r.lastID)
	} else {
		// If no specific image to clear, clear all Kitty images
		control = "a=d"
	}

	output := fmt.Sprintf("\x1b_G%s\x1b\\", control)
	output = wrapTmuxPassthrough(output)
	_, err := io.WriteString(os.Stdout, output)
	return err
}

// generateUnicodePlaceholders generates Unicode placeholders for virtual images
func (r *KittyRenderer) generateUnicodePlaceholders(imageID uint32, cols, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}

	// Create placeholder area using the existing functions
	area := CreatePlaceholderArea(imageID, uint16(rows), uint16(cols))

	// Render with proper image ID color encoding
	return RenderPlaceholderAreaWithImageID(area, imageID)
}

// AnimateImages creates an animation sequence from a list of image IDs
func (r *KittyRenderer) AnimateImages(imageIDs []string, delayMs int, loops int) error {
	if len(imageIDs) == 0 {
		return fmt.Errorf("no image IDs provided for animation")
	}

	// Build animation control sequence
	control := fmt.Sprintf("a=a,i=%s,d=%d,l=%d",
		strings.Join(imageIDs, ":"), delayMs, loops)

	output := fmt.Sprintf("\x1b_G%s,q=1\x1b\\", control)
	output = wrapTmuxPassthrough(output)

	_, err := io.WriteString(os.Stdout, output)
	return err
}

// PlaceImage positions a previously transferred image at specific coordinates
func (r *KittyRenderer) PlaceImage(imageID string, xCells, yCells, zIndex int) error {
	if imageID == "" {
		return fmt.Errorf("image ID is required for placement")
	}

	// Build placement control sequence
	control := fmt.Sprintf("a=p,i=%s,x=%d,y=%d", imageID, xCells, yCells)

	if zIndex > 0 {
		control += fmt.Sprintf(",z=%d", zIndex)
	}

	output := fmt.Sprintf("\x1b_G%s,q=1\x1b\\", control)
	output = wrapTmuxPassthrough(output)

	_, err := io.WriteString(os.Stdout, output)
	return err
}

// SendFile optimizes transfer by sending file path instead of data when possible
func (r *KittyRenderer) SendFile(filePath string, opts RenderOptions) error {
	if filePath == "" {
		return fmt.Errorf("file path is required")
	}

	// Generate unique image ID using atomic counter
	imageID := atomic.AddUint32(&globalKittyImageID, 1)
	r.lastID = imageID

	// Build control parameters for file transfer (with quiet mode)
	control := fmt.Sprintf("a=T,f=100,t=f,i=%d,q=1", imageID)

	// Add z-index if specified
	if opts.ZIndex > 0 {
		control += fmt.Sprintf(",z=%d", opts.ZIndex)
	}

	// Add virtual placement if specified
	if opts.Virtual {
		control += ",U=1"
	}

	// Encode file path
	encodedPath := base64.StdEncoding.EncodeToString([]byte(filePath))

	// Build the escape sequence (quiet mode already included in control)
	output := fmt.Sprintf("\x1b_G%s;%s\x1b\\", control, encodedPath)
	output = wrapTmuxPassthrough(output)

	_, err := io.WriteString(os.Stdout, output)
	return err
}

// ClearVirtualImage explicitly deletes a virtual image by ID
func (r *KittyRenderer) ClearVirtualImage(imageID string) error {
	if imageID == "" {
		return fmt.Errorf("image ID is required for virtual image clearing")
	}

	// Build delete control sequence specifically for virtual images
	control := fmt.Sprintf("a=d,d=i,i=%s", imageID)
	output := fmt.Sprintf("\x1b_G%s,q=1\x1b\\", control)
	output = wrapTmuxPassthrough(output)

	_, err := io.WriteString(os.Stdout, output)
	return err
}

// Unicode placeholder character - using U+10EEEE as suggested by Kitty maintainer
const PLACEHOLDER_CHAR = "\U0010EEEE"

// Official Kitty diacriticals array (297 characters) from the Kitty graphics protocol specification
// This is the complete set of combining diacritical marks used by Kitty for encoding row/column positions
var kittyDiacritics = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F, 0x0346, 0x034A,
	0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357, 0x035B, 0x0363, 0x0364, 0x0365,
	0x0366, 0x0367, 0x0368, 0x0369, 0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F,
	0x0483, 0x0484, 0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
	0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1, 0x05A8, 0x05A9,
	0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611, 0x0612, 0x0613, 0x0614, 0x0615,
	0x0616, 0x0617, 0x0653, 0x0654, 0x0657, 0x0658, 0x0659, 0x065A, 0x065B, 0x065D,
	0x065E, 0x06D6, 0x06D7, 0x06D8, 0x06D9, 0x06DA, 0x06DB, 0x06DC, 0x06DF, 0x06E0,
	0x06E1, 0x06E2, 0x06E4, 0x06E7, 0x06E8, 0x06EB, 0x06EC, 0x0730, 0x0732, 0x0733,
	0x0735, 0x0736, 0x073A, 0x073D, 0x073F, 0x0741, 0x0743, 0x0745, 0x0747, 0x0749,
	0x074A, 0x07EB, 0x07EC, 0x07ED, 0x07EE, 0x07EF, 0x07F0, 0x07F1, 0x07F2, 0x07F3,
	0x0816, 0x0817, 0x0818, 0x0819, 0x081B, 0x081C, 0x081D, 0x081E, 0x081F, 0x0820,
	0x0821, 0x0822, 0x0823, 0x0825, 0x0826, 0x0827, 0x0829, 0x082A, 0x082B, 0x082C,
	0x082D, 0x0951, 0x0952, 0x0953, 0x0954, 0x0F82, 0x0F83, 0x0F86, 0x0F87, 0x135D,
	0x135E, 0x135F, 0x17DD, 0x193A, 0x1A17, 0x1A75, 0x1A76, 0x1A77, 0x1A78, 0x1A79,
	0x1A7A, 0x1A7B, 0x1A7C, 0x1AB0, 0x1AB1, 0x1AB2, 0x1AB3, 0x1AB4, 0x1AB5, 0x1AB6,
	0x1AB7, 0x1AB8, 0x1AB9, 0x1ABA, 0x1ABB, 0x1ABC, 0x1ABD, 0x1B6B, 0x1B6C, 0x1B6D,
	0x1B6E, 0x1B6F, 0x1B70, 0x1B71, 0x1B72, 0x1B73, 0x1CD0, 0x1CD1, 0x1CD2, 0x1CD4,
	0x1CD5, 0x1CD6, 0x1CD7, 0x1CD8, 0x1CD9, 0x1CDA, 0x1CDB, 0x1CDC, 0x1CDD, 0x1CDE,
	0x1CDF, 0x1CE0, 0x1CE2, 0x1CE3, 0x1CE4, 0x1CE5, 0x1CE6, 0x1CE7, 0x1CE8, 0x1CED,
	0x1CF4, 0x1CF8, 0x1CF9, 0x1DC0, 0x1DC1, 0x1DC2, 0x1DC3, 0x1DC4, 0x1DC5, 0x1DC6,
	0x1DC7, 0x1DC8, 0x1DC9, 0x1DCA, 0x1DCB, 0x1DCC, 0x1DCD, 0x1DCE, 0x1DCF, 0x1DD0,
	0x1DD1, 0x1DD2, 0x1DD3, 0x1DD4, 0x1DD5, 0x1DD6, 0x1DD7, 0x1DD8, 0x1DD9, 0x1DDA,
	0x1DDB, 0x1DDC, 0x1DDD, 0x1DDE, 0x1DDF, 0x1DE0, 0x1DE1, 0x1DE2, 0x1DE3, 0x1DE4,
	0x1DE5, 0x1DE6, 0x1DE7, 0x1DE8, 0x1DE9, 0x1DEA, 0x1DEB, 0x1DEC, 0x1DED, 0x1DEE,
	0x1DEF, 0x1DF0, 0x1DF1, 0x1DF2, 0x1DF3, 0x1DF4, 0x1DF5, 0x1DFB, 0x1DFC, 0x1DFD,
	0x1DFE, 0x1DFF, 0x20D0, 0x20D1, 0x20D2, 0x20D3, 0x20D4, 0x20D5, 0x20D6, 0x20D7,
	0x20D8, 0x20D9, 0x20DA, 0x20DB, 0x20DC, 0x20E1, 0x20E5, 0x20E6, 0x20E7, 0x20E8,
	0x20E9, 0x20EA, 0x20EB, 0x20EC, 0x20ED, 0x20EE, 0x20EF, 0x20F0, 0x2CEF, 0x2CF0,
	0x2CF1, 0x2D7F, 0x2DE0, 0x2DE1, 0x2DE2, 0x2DE3, 0x2DE4, 0x2DE5, 0x2DE6, 0x2DE7,
	0x2DE8, 0x2DE9, 0x2DEA, 0x2DEB, 0x2DEC, 0x2DED, 0x2DEE, 0x2DEF, 0x2DF0, 0x2DF1,
	0x2DF2, 0x2DF3, 0x2DF4, 0x2DF5, 0x2DF6, 0x2DF7, 0x2DF8, 0x2DF9, 0x2DFA, 0x2DFB,
	0x2DFC, 0x2DFD, 0x2DFE, 0x2DFF, 0x302A, 0x302B, 0x302C, 0x302D, 0x3099, 0x309A,
	0xA66F, 0xA674, 0xA675, 0xA676, 0xA677, 0xA678, 0xA679, 0xA67A, 0xA67B, 0xA67C,
	0xA67D, 0xA69E, 0xA69F, 0xA6F0, 0xA6F1, 0xA806, 0xA8C4, 0xA8E0, 0xA8E1, 0xA8E2,
	0xA8E3, 0xA8E4, 0xA8E5, 0xA8E6, 0xA8E7, 0xA8E8, 0xA8E9, 0xA8EA, 0xA8EB, 0xA8EC,
	0xA8ED, 0xA8EE, 0xA8EF, 0xA8F0, 0xA8F1, 0xAAB0, 0xAAB2, 0xAAB3, 0xAAB4, 0xAAB7,
	0xAAB8, 0xAABE, 0xAABF, 0xAAC1, 0xAAEC, 0xAAED, 0xAAF6, 0xABE5, 0xABE8, 0xABED,
	0xFB1E, 0xFE00, 0xFE01, 0xFE02, 0xFE03, 0xFE04, 0xFE05, 0xFE06, 0xFE07, 0xFE08,
	0xFE09, 0xFE0A, 0xFE0B, 0xFE0C, 0xFE0D, 0xFE0E, 0xFE0F, 0xFE20, 0xFE21, 0xFE22,
	0xFE23, 0xFE24, 0xFE25, 0xFE26, 0xFE27, 0xFE28, 0xFE29, 0xFE2A, 0xFE2B, 0xFE2C,
	0xFE2D, 0xFE2E, 0xFE2F, 0x101FD, 0x102E0, 0x10376, 0x10377, 0x10378, 0x10379,
	0x1037A, 0x10A0D, 0x10A0F, 0x10A38, 0x10A39, 0x10A3A, 0x10A3F, 0x10AE5, 0x10AE6,
	0x11001, 0x11038, 0x11039, 0x1103A, 0x1103B, 0x1103C, 0x1103D, 0x1103E, 0x1103F,
	0x11040, 0x11041, 0x11042, 0x11043, 0x11044, 0x11045, 0x11046, 0x1107F, 0x110B3,
	0x110B4, 0x110B5, 0x110B6, 0x110B9, 0x110BA, 0x11100, 0x11101, 0x11102, 0x11127,
	0x11128, 0x11129, 0x1112A, 0x1112B, 0x1112D, 0x1112E, 0x1112F, 0x11130, 0x11131,
	0x11132, 0x11133, 0x11134, 0x11173, 0x11180, 0x11181, 0x111B6, 0x111B7, 0x111B8,
	0x111B9, 0x111BA, 0x111BB, 0x111BC, 0x111BD, 0x111BE, 0x111CA, 0x111CB, 0x111CC,
	0x1122F, 0x11230, 0x11231, 0x11234, 0x11236, 0x11237, 0x1123E, 0x112DF, 0x112E3,
	0x112E4, 0x112E5, 0x112E6, 0x112E7, 0x112E8, 0x112E9, 0x112EA, 0x11300, 0x11301,
	0x1133C, 0x1133E, 0x11340, 0x11366, 0x11367, 0x11368, 0x11369, 0x1136A, 0x1136B,
	0x1136C, 0x11370, 0x11371, 0x11372, 0x11373, 0x11374, 0x11438, 0x11439, 0x1143A,
	0x1143B, 0x1143C, 0x1143D, 0x1143E, 0x1143F, 0x11442, 0x11443, 0x11444, 0x11446,
	0x114B3, 0x114B4, 0x114B5, 0x114B6, 0x114B7, 0x114B8, 0x114BA, 0x114BF, 0x114C0,
	0x114C2, 0x114C3, 0x115B2, 0x115B3, 0x115B4, 0x115B5, 0x115BC, 0x115BD, 0x115BF,
	0x115C0, 0x115DC, 0x115DD, 0x11633, 0x11634, 0x11635, 0x11636, 0x11637, 0x11638,
	0x11639, 0x1163A, 0x1163D, 0x1163F, 0x11640, 0x116AB, 0x116AD, 0x116B0, 0x116B1,
	0x116B2, 0x116B3, 0x116B4, 0x116B5, 0x116B7, 0x1171D, 0x1171E, 0x1171F, 0x11722,
	0x11723, 0x11724, 0x11725, 0x11727, 0x11728, 0x11729, 0x1172A, 0x1172B, 0x11C30,
	0x11C31, 0x11C32, 0x11C33, 0x11C34, 0x11C35, 0x11C36, 0x11C38, 0x11C39, 0x11C3A,
	0x11C3B, 0x11C3C, 0x11C3D, 0x11C3F, 0x11C92, 0x11C93, 0x11C94, 0x11C95, 0x11C96,
	0x11C97, 0x11C98, 0x11C99, 0x11C9A, 0x11C9B, 0x11C9C, 0x11C9D, 0x11C9E, 0x11C9F,
	0x11CA0, 0x11CA1, 0x11CA2, 0x11CA3, 0x11CA4, 0x11CA5, 0x11CA6, 0x11CA7, 0x11CAA,
	0x11CAB, 0x11CAC, 0x11CAD, 0x11CAE, 0x11CAF, 0x11CB0, 0x11CB2, 0x11CB3, 0x11CB5,
	0x11CB6, 0x16AF0, 0x16AF1, 0x16AF2, 0x16AF3, 0x16AF4, 0x16B30, 0x16B31, 0x16B32,
	0x16B33, 0x16B34, 0x16B35, 0x16B36, 0x1BC9D, 0x1BC9E, 0x1D167, 0x1D168, 0x1D169,
	0x1D17B, 0x1D17C, 0x1D17D, 0x1D17E, 0x1D17F, 0x1D180, 0x1D181, 0x1D182, 0x1D185,
	0x1D186, 0x1D187, 0x1D188, 0x1D189, 0x1D18A, 0x1D18B, 0x1D1AA, 0x1D1AB, 0x1D1AC,
	0x1D1AD, 0x1D242, 0x1D243, 0x1D244, 0x1E000, 0x1E001, 0x1E002, 0x1E003, 0x1E004,
	0x1E005, 0x1E006, 0x1E008, 0x1E009, 0x1E00A, 0x1E00B, 0x1E00C, 0x1E00D, 0x1E00E,
	0x1E00F, 0x1E010, 0x1E011, 0x1E012, 0x1E013, 0x1E014, 0x1E015, 0x1E016, 0x1E017,
	0x1E018, 0x1E01B, 0x1E01C, 0x1E01D, 0x1E01E, 0x1E01F, 0x1E020, 0x1E021, 0x1E023,
	0x1E024, 0x1E026, 0x1E027, 0x1E028, 0x1E029, 0x1E02A, 0x1E8D0, 0x1E8D1, 0x1E8D2,
	0x1E8D3, 0x1E8D4, 0x1E8D5, 0x1E8D6,
}

// diacritic returns the diacritic character for a given position
func diacritic(pos uint16) rune {
	if pos >= uint16(len(kittyDiacritics)) {
		return kittyDiacritics[0] // fallback to first diacritic for overflow
	}
	return kittyDiacritics[pos]
}

// CreatePlaceholder generates a Unicode placeholder for the given image position
// placeholder_char + row_diacritic + column_diacritic + id_extra_diacritic
func CreatePlaceholder(row, column uint16, id_extra byte) string {
	var builder strings.Builder

	// Add the placeholder character
	builder.WriteString(PLACEHOLDER_CHAR)

	// Add diacritical marks for row and column
	builder.WriteRune(diacritic(row))
	builder.WriteRune(diacritic(column))
	// Add diacritic for the extra ID byte
	builder.WriteRune(diacritic(uint16(id_extra)))

	return builder.String()
}

// CreatePlaceholderArea generates a grid of placeholders for an image
func CreatePlaceholderArea(imageID uint32, rows, columns uint16) [][]string {
	id_extra := byte(imageID >> 24)
	area := make([][]string, rows)
	for r := range rows {
		area[r] = make([]string, columns)
		for c := range columns {
			// Use 0-based indexing for row and column positions
			area[r][c] = CreatePlaceholder(r, c, id_extra)
		}
	}
	return area
}

// RenderPlaceholderAreaWithImageID converts a placeholder area to a string with proper color encoding
// The image ID is encoded in the ANSI foreground color as per Kitty specification
func RenderPlaceholderAreaWithImageID(area [][]string, imageID uint32) string {
	var builder strings.Builder

	// Set color once at the beginning to encode image ID
	// Use simple color encoding - image ID modulo 256 for each RGB component
	colorCode := imageID & 0xFFFFFF
	r := (colorCode >> 16) & 0xFF
	g := (colorCode >> 8) & 0xFF
	b := colorCode & 0xFF

	// Set foreground color to encode image ID
	builder.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b))

	for i, row := range area {
		for _, placeholder := range row {
			builder.WriteString(placeholder)
		}
		if i < len(area)-1 {
			builder.WriteString("\n")
		}
	}

	// Reset color at the end
	builder.WriteString("\x1b[39m")

	return builder.String()
}

/* old utils */

var ErrEmptyResponse = fmt.Errorf("empty response")

type KittyResponse struct {
	ID      string
	Message string
}

func parseResponse(in []byte) (*KittyResponse, error) {
	if len(in) == 0 {
		return nil, ErrEmptyResponse
	}
	var resp KittyResponse
	in = bytes.Trim(in, "\x00")
	in = bytes.TrimSuffix(in, []byte("\x1b\\"))
	in = bytes.TrimPrefix(in, []byte("\x1b_G"))
	for field := range bytes.SplitSeq(in, []byte(";")) {
		kv := bytes.Split(field, []byte("="))
		if len(kv) != 2 {
			resp.Message = string(field)
			continue
		}
		switch string(kv[0]) {
		case "i":
			resp.ID = string(kv[1])
		default:
			return nil, fmt.Errorf("unknown field: %s", string(kv[0]))
		}
	}
	return &resp, nil
}
