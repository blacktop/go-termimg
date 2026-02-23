package termimg

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageWidgetRenderVirtualUsesInheritedPlaceholders(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	widget := NewImageWidget(New(img)).
		SetProtocol(Kitty).
		SetVirtual(true).
		SetSize(3, 2).
		SetPosition(4, 5)

	output, err := widget.RenderVirtual()
	require.NoError(t, err)
	require.NotZero(t, widget.imageID, "virtual render should capture the kitty image ID")

	assert.NotContains(t, output, "\x1b[s", "virtual render should not save cursor position")
	assert.NotContains(t, output, "\x1b[u", "virtual render should not restore cursor position")
	assert.Contains(t, output, "\x1b[6;5H")
	assert.Contains(t, output, "\x1b[7;5H")

	idExtra := byte(widget.imageID >> 24)
	for row := range 2 {
		assert.Contains(t, output, CreatePlaceholder(uint16(row), 0, idExtra)+PLACEHOLDER_CHAR+PLACEHOLDER_CHAR)
		assert.NotContains(t, output, CreatePlaceholder(uint16(row), 1, idExtra))
		assert.NotContains(t, output, CreatePlaceholder(uint16(row), 2, idExtra))
	}
}
