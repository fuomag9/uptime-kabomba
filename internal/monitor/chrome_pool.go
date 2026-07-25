package monitor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"

	"github.com/chromedp/chromedp"
)

// ErrPoolSaturated is returned by AcquireContext when every pool slot is
// currently in use.
var ErrPoolSaturated = errors.New("chrome pool is saturated")

// ChromePool manages a pool of Chrome browser contexts for concurrent page checks.
type ChromePool struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	sem         chan struct{}
	mu          sync.Mutex
	closed      bool
}

// NewChromePool creates a new Chrome pool with the given binary path and concurrency limit.
// If chromePath is empty, chromedp auto-detects the Chrome/Chromium binary.
func NewChromePool(chromePath string, maxConcurrent int) (*ChromePool, error) {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
	)

	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// Verify Chrome works by creating and immediately closing a test context
	testCtx, testCancel := chromedp.NewContext(allocCtx)
	if err := chromedp.Run(testCtx); err != nil {
		testCancel()
		allocCancel()
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}
	testCancel()

	return &ChromePool{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		sem:         make(chan struct{}, maxConcurrent),
	}, nil
}

// AcquireContext acquires a semaphore slot and returns a new Chrome tab context.
// The caller must call the returned cancel function and then Release() when done.
//
// It does NOT queue when the pool is full: it returns ErrPoolSaturated
// immediately. The pool is a small, process-global resource shared by every
// page_change monitor; letting checks queue for a slot meant a handful of
// monitors hitting a slow/hanging URL could tie up every other page_change
// monitor's goroutine for its entire check timeout. Failing fast lets the
// caller report "skipped, pool busy" and try again next interval instead.
func (p *ChromePool) AcquireContext(ctx context.Context) (context.Context, context.CancelFunc, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, nil, fmt.Errorf("chrome pool is closed")
	}
	p.mu.Unlock()

	select {
	case p.sem <- struct{}{}:
	default:
		return nil, nil, ErrPoolSaturated
	}

	tabCtx, tabCancel := chromedp.NewContext(p.allocCtx)

	// Apply the caller's deadline/timeout to the tab context
	if deadline, ok := ctx.Deadline(); ok {
		var timeoutCancel context.CancelFunc
		tabCtx, timeoutCancel = context.WithDeadline(tabCtx, deadline)
		origCancel := tabCancel
		tabCancel = func() {
			timeoutCancel()
			origCancel()
		}
	}

	return tabCtx, tabCancel, nil
}

// Release releases a semaphore slot back to the pool.
func (p *ChromePool) Release() {
	<-p.sem
}

// Close shuts down the Chrome pool and all browser processes.
func (p *ChromePool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		p.closed = true
		p.allocCancel()
	}
}

// Available checks if a Chrome/Chromium binary is available on the system.
func Available(chromePath string) bool {
	if chromePath != "" {
		_, err := exec.LookPath(chromePath)
		return err == nil
	}
	// Check common Chrome/Chromium binary names
	for _, name := range []string{"chromium-browser", "chromium", "google-chrome", "google-chrome-stable", "chrome"} {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	return false
}
