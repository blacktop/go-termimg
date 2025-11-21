package termimg

import (
	"fmt"
	"sync"
	"time"
)

// RenderOutcome describes the result of rendering a widget into a viewport.
// Pending is true when an async render is in-flight and the returned Output
// may still be from a previous size. Skipped indicates the viewport was too
// small to render anything safely.
type RenderOutcome struct {
	Output   string
	Width    int
	Height   int
	Duration time.Duration
	Err      error
	Pending  bool
	Skipped  bool
}

// AsyncWorkerOptions configures the render worker.
type AsyncWorkerOptions struct {
	Workers int // number of goroutines to use; defaults to 1
	Queue   int // size of the request/result buffers; defaults to 1 (latest wins)
}

// renderRequest is the minimal set of fields needed to reproduce a render.
type renderRequest struct {
	width    int
	height   int
	protocol Protocol
	scale    ScaleMode
	virtual  bool
	zIndex   int
}

// AsyncRenderWorker renders images on background goroutines so callers can
// keep UI loops responsive. When the queue is full, newer requests replace
// older ones to always prioritise the latest viewport.
type AsyncRenderWorker struct {
	base          *Image
	reqCh         chan renderRequest
	resCh         chan RenderOutcome
	stopCh        chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
	lastRequested renderRequest
	lastResult    RenderOutcome
}

// NewAsyncRenderWorker starts a worker for the provided image.
func NewAsyncRenderWorker(img *Image, opts AsyncWorkerOptions) *AsyncRenderWorker {
	workers := opts.Workers
	if workers <= 0 {
		workers = 1
	}

	queue := opts.Queue
	if queue <= 0 {
		queue = 1
	}

	w := &AsyncRenderWorker{
		base:   cloneImage(img),
		reqCh:  make(chan renderRequest, queue),
		resCh:  make(chan RenderOutcome, queue),
		stopCh: make(chan struct{}),
	}

	for i := 0; i < workers; i++ {
		w.wg.Add(1)
		go w.loop()
	}

	return w
}

// Close stops all worker goroutines.
func (w *AsyncRenderWorker) Close() {
	close(w.stopCh)
	w.wg.Wait()
}

// Schedule enqueues a render request. If an identical request is already the
// most recent, it is skipped. When the queue is full the oldest pending
// request is dropped to keep the pipeline current.
func (w *AsyncRenderWorker) Schedule(req renderRequest) {
	w.mu.Lock()
	if sameRequest(req, w.lastRequested) {
		w.mu.Unlock()
		return
	}
	w.lastRequested = req
	w.mu.Unlock()

	select {
	case w.reqCh <- req:
	default:
		<-w.reqCh
		w.reqCh <- req
	}
}

// TryLatest returns the newest completed render, if any. It drains the result
// buffer to always surface the most recent output.
func (w *AsyncRenderWorker) TryLatest() (RenderOutcome, bool) {
	for {
		select {
		case res := <-w.resCh:
			w.mu.Lock()
			w.lastResult = res
			w.mu.Unlock()
		default:
			w.mu.Lock()
			res := w.lastResult
			has := res.Output != "" || res.Err != nil || res.Skipped
			w.mu.Unlock()
			return res, has
		}
	}
}

// loop consumes render requests and executes them.
func (w *AsyncRenderWorker) loop() {
	defer w.wg.Done()

	for {
		select {
		case req := <-w.reqCh:
			res := renderOnce(w.base, req)
			select {
			case w.resCh <- res:
			default:
				<-w.resCh
				w.resCh <- res
			}
		case <-w.stopCh:
			return
		}
	}
}

// StatefulImageWidget tracks viewport size and re-renders only when needed.
// It can operate synchronously or with an AsyncRenderWorker for background
// rendering.
type StatefulImageWidget struct {
	image     *Image
	protocol  Protocol
	scaleMode ScaleMode
	minWidth  int
	minHeight int
	virtual   bool
	zIndex    int

	worker *AsyncRenderWorker

	mu         sync.Mutex
	lastTarget renderRequest
	lastResult RenderOutcome
}

// NewStatefulImageWidget creates a widget that can adapt to changing viewports.
func NewStatefulImageWidget(img *Image) *StatefulImageWidget {
	return &StatefulImageWidget{
		image:     img,
		protocol:  Auto,
		scaleMode: ScaleFit,
		minWidth:  1,
		minHeight: 1,
	}
}

// SetProtocol overrides the protocol used for rendering.
func (w *StatefulImageWidget) SetProtocol(protocol Protocol) *StatefulImageWidget {
	w.protocol = protocol
	return w
}

// SetScaleMode controls how the image scales inside the viewport.
func (w *StatefulImageWidget) SetScaleMode(mode ScaleMode) *StatefulImageWidget {
	w.scaleMode = mode
	return w
}

// SetMinimumCells configures the minimum width/height (in cells) required
// before rendering. Calls with smaller viewports will be skipped.
func (w *StatefulImageWidget) SetMinimumCells(minWidth, minHeight int) *StatefulImageWidget {
	if minWidth < 1 {
		minWidth = 1
	}
	if minHeight < 1 {
		minHeight = 1
	}
	w.minWidth = minWidth
	w.minHeight = minHeight
	return w
}

// EnableAsync spins up a background worker. workers<=0 defaults to 1.
func (w *StatefulImageWidget) EnableAsync(workers int) *StatefulImageWidget {
	w.worker = NewAsyncRenderWorker(w.image, AsyncWorkerOptions{Workers: workers})
	return w
}

// WithWorker attaches a caller-supplied worker (useful for sharing across widgets).
func (w *StatefulImageWidget) WithWorker(worker *AsyncRenderWorker) *StatefulImageWidget {
	w.worker = worker
	return w
}

// SetVirtual toggles Kitty virtual placement support.
func (w *StatefulImageWidget) SetVirtual(virtual bool) *StatefulImageWidget {
	w.virtual = virtual
	return w
}

// SetZIndex sets Kitty z-index ordering.
func (w *StatefulImageWidget) SetZIndex(z int) *StatefulImageWidget {
	w.zIndex = z
	return w
}

// Close stops the attached worker, if any.
func (w *StatefulImageWidget) Close() {
	if w.worker != nil {
		w.worker.Close()
	}
}

// RenderInto renders the widget into a viewport of width x height cells. When
// async is enabled, Pending will be true until the worker finishes a render that
// matches the current target size.
func (w *StatefulImageWidget) RenderInto(width, height int) RenderOutcome {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.image == nil {
		return RenderOutcome{Err: fmt.Errorf("no image configured")}
	}

	targetW, targetH := targetSize(width, height, w.image.Bounds.Dx(), w.image.Bounds.Dy(), w.scaleMode)
	if targetW < w.minWidth || targetH < w.minHeight {
		return RenderOutcome{Skipped: true, Width: targetW, Height: targetH}
	}

	req := renderRequest{
		width:    targetW,
		height:   targetH,
		protocol: w.protocol,
		scale:    w.scaleMode,
		virtual:  w.virtual,
		zIndex:   w.zIndex,
	}

	// If nothing changed and we already rendered once, reuse.
	if sameRequest(req, w.lastTarget) && !w.lastResult.Pending && w.lastResult.Output != "" && w.lastResult.Err == nil {
		return w.lastResult
	}

	w.lastTarget = req

	if w.worker != nil {
		w.worker.Schedule(req)
		if res, ok := w.worker.TryLatest(); ok {
			w.lastResult = res
		}

		outcome := w.lastResult
		if outcome.Width != targetW || outcome.Height != targetH {
			outcome.Pending = true
		}
		return outcome
	}

	res := renderOnce(w.image, req)
	w.lastResult = res
	return res
}

// targetSize computes the desired render size in character cells, clamped to
// the viewport while preserving aspect ratios for fit/auto modes.
func targetSize(viewW, viewH, imgW, imgH int, mode ScaleMode) (int, int) {
	if viewW <= 0 || viewH <= 0 || imgW <= 0 || imgH <= 0 {
		return 0, 0
	}

	switch mode {
	case ScaleFill, ScaleStretch:
		return viewW, viewH
	case ScaleNone:
		if imgW > viewW {
			imgW = viewW
		}
		if imgH > viewH {
			imgH = viewH
		}
		return imgW, imgH
	default: // ScaleFit and ScaleAuto behave like fit at the widget level
		imgRatio := float64(imgW) / float64(imgH)
		targetW := viewW
		targetH := int(float64(targetW) / imgRatio)
		if targetH > viewH {
			targetH = viewH
			targetW = int(float64(targetH) * imgRatio)
		}
		if targetW < 1 {
			targetW = 1
		}
		if targetH < 1 {
			targetH = 1
		}
		return targetW, targetH
	}
}

// renderOnce performs a single render using the provided request parameters.
func renderOnce(base *Image, req renderRequest) RenderOutcome {
	if base == nil {
		return RenderOutcome{Err: fmt.Errorf("no image available"), Width: req.width, Height: req.height}
	}

	img := cloneImage(base)
	if img == nil {
		return RenderOutcome{Err: fmt.Errorf("failed to clone image"), Width: req.width, Height: req.height}
	}

	img = img.Width(req.width).Height(req.height).Scale(req.scale).
		Protocol(req.protocol).Virtual(req.virtual).ZIndex(req.zIndex)

	start := time.Now()
	output, err := img.Render()
	return RenderOutcome{
		Output:   output,
		Width:    req.width,
		Height:   req.height,
		Duration: time.Since(start),
		Err:      err,
	}
}

// sameRequest determines if two renderRequest values are identical.
func sameRequest(a, b renderRequest) bool {
	return a.width == b.width && a.height == b.height && a.protocol == b.protocol && a.scale == b.scale && a.virtual == b.virtual && a.zIndex == b.zIndex
}

// cloneImage duplicates the Image metadata so we can modify it per render
// without mutating the caller's instance.
func cloneImage(img *Image) *Image {
	if img == nil {
		return nil
	}
	copyImg := *img
	copyImg.renderer = nil
	return &copyImg
}
