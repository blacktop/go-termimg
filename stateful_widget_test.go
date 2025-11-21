package termimg

import (
	"image"
	"testing"
	"time"
)

func TestStatefulWidgetSkipsForSmallViewport(t *testing.T) {
	t.Setenv("TERMIMG_BYPASS_DETECTION", "halfblocks")

	img := New(image.NewRGBA(image.Rect(0, 0, 10, 10)))
	widget := NewStatefulImageWidget(img).SetMinimumCells(2, 2).SetProtocol(Halfblocks)

	outcome := widget.RenderInto(1, 1)
	if !outcome.Skipped {
		t.Fatalf("expected render to be skipped for small viewport")
	}
}

func TestStatefulWidgetFitsWithinViewport(t *testing.T) {
	t.Setenv("TERMIMG_BYPASS_DETECTION", "halfblocks")

	img := New(image.NewRGBA(image.Rect(0, 0, 50, 30)))
	widget := NewStatefulImageWidget(img).SetProtocol(Halfblocks)

	outcome := widget.RenderInto(10, 4)
	if outcome.Err != nil {
		t.Fatalf("render failed: %v", outcome.Err)
	}
	if outcome.Skipped {
		t.Fatalf("render unexpectedly skipped")
	}
	if outcome.Width > 10 || outcome.Height > 4 {
		t.Fatalf("render size exceeded viewport: %dx%d", outcome.Width, outcome.Height)
	}
	if outcome.Output == "" {
		t.Fatalf("expected non-empty render output")
	}
}

func TestAsyncRenderWorkerProducesLatestResult(t *testing.T) {
	t.Setenv("TERMIMG_BYPASS_DETECTION", "halfblocks")

	img := New(image.NewRGBA(image.Rect(0, 0, 20, 20)))
	worker := NewAsyncRenderWorker(img, AsyncWorkerOptions{Workers: 1})
	t.Cleanup(worker.Close)

	worker.Schedule(renderRequest{width: 3, height: 3, protocol: Halfblocks, scale: ScaleFit})

	res := waitForResult(t, worker, 2*time.Second)
	if res.Err != nil {
		t.Fatalf("worker render failed: %v", res.Err)
	}
	if res.Width != 3 || res.Height != 3 {
		t.Fatalf("unexpected render dimensions %dx%d", res.Width, res.Height)
	}
	if res.Output == "" {
		t.Fatalf("expected render output")
	}
}

func TestStatefulWidgetAsyncPendingThenReady(t *testing.T) {
	t.Setenv("TERMIMG_BYPASS_DETECTION", "halfblocks")

	img := New(image.NewRGBA(image.Rect(0, 0, 16, 16)))
	widget := NewStatefulImageWidget(img).
		SetProtocol(Halfblocks).
		EnableAsync(1)
	t.Cleanup(widget.Close)

	first := widget.RenderInto(8, 4)
	if first.Err != nil {
		t.Fatalf("initial render errored: %v", first.Err)
	}
	if first.Output == "" && !first.Pending {
		t.Fatalf("expected pending or rendered output on first call")
	}

	res := waitForResult(t, widget.worker, 2*time.Second)
	if res.Err != nil {
		t.Fatalf("async render failed: %v", res.Err)
	}

	second := widget.RenderInto(8, 4)
	if second.Pending {
		t.Fatalf("render still pending after worker completion")
	}
	if second.Output == "" {
		t.Fatalf("expected final render output")
	}
}

func waitForResult(t *testing.T, worker *AsyncRenderWorker, timeout time.Duration) RenderOutcome {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if res, ok := worker.TryLatest(); ok {
			return res
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for worker result")
	return RenderOutcome{}
}
