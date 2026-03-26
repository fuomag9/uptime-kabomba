package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"gorm.io/gorm"
)

// PageChangeSnapshot represents a stored snapshot for page change detection.
type PageChangeSnapshot struct {
	ID             int       `json:"id" gorm:"primaryKey;autoIncrement"`
	MonitorID      int       `json:"monitor_id" gorm:"not null;index"`
	HeartbeatID    *int      `json:"heartbeat_id"`
	ScreenshotPath string    `json:"screenshot_path"`
	BaselinePath   string    `json:"baseline_path"`
	DiffPath       string    `json:"diff_path"`
	HTMLHash       string    `json:"html_hash"`
	RuntimeMetrics string    `json:"runtime_metrics" gorm:"type:jsonb"`
	ChangeScore    float64   `json:"change_score"`
	ImageScore     float64   `json:"image_score"`
	HTMLScore      float64   `json:"html_score"`
	RuntimeScore   float64   `json:"runtime_score"`
	IsBaseline     bool      `json:"is_baseline"`
	CreatedAt      time.Time `json:"created_at"`
}

func (PageChangeSnapshot) TableName() string {
	return "page_change_snapshots"
}

// PageChangeMonitor implements page change detection using headless Chrome.
type PageChangeMonitor struct {
	pool        *ChromePool
	db          *gorm.DB
	storagePath string
}

// NewPageChangeMonitor creates a new PageChangeMonitor.
func NewPageChangeMonitor(pool *ChromePool, db *gorm.DB, storagePath string) *PageChangeMonitor {
	return &PageChangeMonitor{
		pool:        pool,
		db:          db,
		storagePath: storagePath,
	}
}

func (p *PageChangeMonitor) Name() string {
	return "page_change"
}

func (p *PageChangeMonitor) Validate(monitor *Monitor) error {
	if monitor.URL == "" {
		return fmt.Errorf("URL is required for page change monitor")
	}

	if !strings.HasPrefix(monitor.URL, "http://") && !strings.HasPrefix(monitor.URL, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	// SSRF protection
	cfg := GetConfig()
	ssrfProtection := NewSSRFProtection(cfg.AllowPrivateIPs, cfg.AllowMetadataEndpoints)
	if err := ssrfProtection.ValidateURL(monitor.URL); err != nil {
		return fmt.Errorf("URL validation failed: %w", err)
	}

	if monitor.Timeout <= 0 {
		monitor.Timeout = 30
	}
	if monitor.Interval <= 0 {
		monitor.Interval = 300 // 5 minutes default for page change (heavier check)
	}

	// Validate config fields
	threshold := getConfigFloat(monitor, "change_threshold", 0.1)
	if threshold < 0 || threshold > 1 {
		return fmt.Errorf("change_threshold must be between 0 and 1")
	}

	waitTime := getConfigInt(monitor, "wait_time", 3000)
	if waitTime < 0 || waitTime > 30000 {
		return fmt.Errorf("wait_time must be between 0 and 30000 milliseconds")
	}

	vw := getConfigInt(monitor, "viewport_width", 1920)
	if vw < 320 || vw > 3840 {
		return fmt.Errorf("viewport_width must be between 320 and 3840")
	}

	vh := getConfigInt(monitor, "viewport_height", 1080)
	if vh < 240 || vh > 2160 {
		return fmt.Errorf("viewport_height must be between 240 and 2160")
	}

	imageWeight := getConfigFloat(monitor, "image_weight", 0.4)
	htmlWeight := getConfigFloat(monitor, "html_weight", 0.3)
	runtimeWeight := getConfigFloat(monitor, "runtime_weight", 0.3)
	weightSum := imageWeight + htmlWeight + runtimeWeight
	if math.Abs(weightSum-1.0) > 0.05 {
		return fmt.Errorf("heuristic weights must sum to approximately 1.0 (got %.2f)", weightSum)
	}

	return nil
}

func (p *PageChangeMonitor) Check(ctx context.Context, monitor *Monitor) (*Heartbeat, error) {
	heartbeat := &Heartbeat{
		MonitorID: monitor.ID,
		Time:      time.Now(),
		Status:    StatusDown,
	}

	start := time.Now()

	// Read config
	waitTime := getConfigInt(monitor, "wait_time", 3000)
	viewportWidth := getConfigInt(monitor, "viewport_width", 1920)
	viewportHeight := getConfigInt(monitor, "viewport_height", 1080)
	ignoreSelectors := getConfigStringSlice(monitor, "ignore_selectors")
	watchSelectors := getConfigStringSlice(monitor, "watch_selectors")
	customJS := getConfigString(monitor, "custom_js", "")
	changeThreshold := getConfigFloat(monitor, "change_threshold", 0.1)
	imageWeight := getConfigFloat(monitor, "image_weight", 0.4)
	htmlWeight := getConfigFloat(monitor, "html_weight", 0.3)
	runtimeWeight := getConfigFloat(monitor, "runtime_weight", 0.3)

	// Acquire Chrome tab
	tabCtx, tabCancel, err := p.pool.AcquireContext(ctx)
	if err != nil {
		heartbeat.Message = fmt.Sprintf("Failed to acquire Chrome context: %v", err)
		return heartbeat, nil
	}
	defer tabCancel()
	defer p.pool.Release()

	// Navigate and capture
	screenshotData, htmlContent, runtimeMetrics, err := p.capturePage(tabCtx, monitor.URL, waitTime, viewportWidth, viewportHeight, ignoreSelectors, watchSelectors, customJS)
	if err != nil {
		heartbeat.Message = fmt.Sprintf("Page capture failed: %v", err)
		return heartbeat, nil
	}

	ping := time.Since(start).Milliseconds()
	heartbeat.Ping = int(ping)

	// Ensure storage directory exists
	monitorDir := screenshotDir(p.storagePath, monitor.ID)
	if err := ensureStorageDir(monitorDir); err != nil {
		heartbeat.Message = fmt.Sprintf("Failed to create storage directory: %v", err)
		return heartbeat, nil
	}

	// Load the latest baseline
	var baseline PageChangeSnapshot
	err = p.db.Where("monitor_id = ? AND is_baseline = ?", monitor.ID, true).
		Order("created_at DESC").
		First(&baseline).Error

	timestamp := time.Now().Unix()
	screenshotRelPath := filepath.Join(fmt.Sprintf("%d", monitor.ID), fmt.Sprintf("%d_current.png", timestamp))
	screenshotAbsPath := filepath.Join(p.storagePath, screenshotRelPath)

	// Save current screenshot to disk
	if err := os.WriteFile(screenshotAbsPath, screenshotData, 0644); err != nil {
		heartbeat.Message = fmt.Sprintf("Failed to save screenshot: %v", err)
		return heartbeat, nil
	}

	htmlHash := hashString(htmlContent)

	// Marshal runtime metrics
	runtimeJSON, _ := json.Marshal(runtimeMetrics)

	// No baseline exists - this is the first run
	if err == gorm.ErrRecordNotFound {
		// Save HTML for future comparisons
		os.WriteFile(screenshotAbsPath+".html", []byte(htmlContent), 0644)

		snapshot := PageChangeSnapshot{
			MonitorID:      monitor.ID,
			ScreenshotPath: screenshotRelPath,
			BaselinePath:   screenshotRelPath,
			HTMLHash:       htmlHash,
			RuntimeMetrics: string(runtimeJSON),
			ChangeScore:    0,
			ImageScore:     0,
			HTMLScore:      0,
			RuntimeScore:   0,
			IsBaseline:     true,
			CreatedAt:      time.Now(),
		}
		p.db.Create(&snapshot)

		heartbeat.Status = StatusUp
		heartbeat.Message = fmt.Sprintf("Baseline captured - %dms", ping)
		return heartbeat, nil
	} else if err != nil {
		heartbeat.Message = fmt.Sprintf("Failed to query baseline: %v", err)
		return heartbeat, nil
	}

	// Load baseline screenshot from disk
	baselineAbsPath := filepath.Join(p.storagePath, baseline.ScreenshotPath)
	baselineScreenshot, err := os.ReadFile(baselineAbsPath)
	if err != nil {
		// Baseline file missing - re-establish baseline
		log.Printf("Baseline screenshot missing for monitor %d, re-establishing", monitor.ID)
		p.db.Model(&PageChangeSnapshot{}).Where("monitor_id = ? AND is_baseline = ?", monitor.ID, true).
			Update("is_baseline", false)

		os.WriteFile(screenshotAbsPath+".html", []byte(htmlContent), 0644)

		snapshot := PageChangeSnapshot{
			MonitorID:      monitor.ID,
			ScreenshotPath: screenshotRelPath,
			BaselinePath:   screenshotRelPath,
			HTMLHash:       htmlHash,
			RuntimeMetrics: string(runtimeJSON),
			IsBaseline:     true,
			CreatedAt:      time.Now(),
		}
		p.db.Create(&snapshot)

		heartbeat.Status = StatusUp
		heartbeat.Message = fmt.Sprintf("Baseline re-established (previous missing) - %dms", ping)
		return heartbeat, nil
	}

	// Load baseline runtime metrics
	var baselineRuntime map[string]interface{}
	if baseline.RuntimeMetrics != "" {
		json.Unmarshal([]byte(baseline.RuntimeMetrics), &baselineRuntime)
	}

	// Compute image score
	imageScore, diffData, err := compareImages(baselineScreenshot, screenshotData)
	if err != nil {
		log.Printf("Image comparison failed for monitor %d: %v", monitor.ID, err)
		imageScore = 0
	}

	// Save diff image if there are differences
	diffRelPath := ""
	if diffData != nil && imageScore > 0 {
		diffRelPath = filepath.Join(fmt.Sprintf("%d", monitor.ID), fmt.Sprintf("%d_diff.png", timestamp))
		diffAbsPath := filepath.Join(p.storagePath, diffRelPath)
		if err := os.WriteFile(diffAbsPath, diffData, 0644); err != nil {
			log.Printf("Failed to save diff image for monitor %d: %v", monitor.ID, err)
		}
	}

	// Compute HTML score
	var htmlScore float64
	if htmlHash != baseline.HTMLHash {
		htmlScore = 1.0 // Default to full change if hash differs

		// Try to load baseline HTML for more granular comparison
		baselineHTMLPath := baselineAbsPath + ".html"
		if baselineHTMLData, err := os.ReadFile(baselineHTMLPath); err == nil {
			htmlScore = compareHTML(string(baselineHTMLData), htmlContent)
		}
	}
	// Save current HTML for future use
	os.WriteFile(screenshotAbsPath+".html", []byte(htmlContent), 0644)

	// Compute runtime score
	runtimeScore := compareRuntime(baselineRuntime, runtimeMetrics)

	// Combined score
	changeScore := imageWeight*imageScore + htmlWeight*htmlScore + runtimeWeight*runtimeScore
	if changeScore > 1.0 {
		changeScore = 1.0
	}

	// Save snapshot
	snapshot := PageChangeSnapshot{
		MonitorID:      monitor.ID,
		ScreenshotPath: screenshotRelPath,
		BaselinePath:   baseline.ScreenshotPath,
		DiffPath:       diffRelPath,
		HTMLHash:       htmlHash,
		RuntimeMetrics: string(runtimeJSON),
		ChangeScore:    changeScore,
		ImageScore:     imageScore,
		HTMLScore:      htmlScore,
		RuntimeScore:   runtimeScore,
		IsBaseline:     false,
		CreatedAt:      time.Now(),
	}
	p.db.Create(&snapshot)

	autoAccept := getConfigBool(monitor, "auto_accept_changes", true)

	// Rotate baseline logic:
	// - Below threshold: always rotate (page is stable, track gradual drift)
	// - Above threshold + auto_accept=true: notify DOWN once, then accept as new baseline
	//   so the monitor recovers to UP and only fires again on the NEXT change
	// - Above threshold + auto_accept=false: keep old baseline, stay DOWN until
	//   page reverts or user manually resets (delete & recreate monitor)
	if changeScore < changeThreshold {
		// Stable - rotate baseline
		p.rotateBaseline(monitor.ID, &snapshot, screenshotAbsPath, htmlContent)

		heartbeat.Status = StatusUp
		heartbeat.Message = fmt.Sprintf("No significant change (score: %.2f) - %dms", changeScore, ping)
	} else if autoAccept {
		// Changed + auto-accept: check if this is the first detection or a repeat.
		// On first detection: report DOWN (triggers notification).
		// On subsequent checks with same content: accept as new baseline, report UP.
		// We detect "repeat" by checking if previous snapshot had the same HTML hash.
		var prevSnapshot PageChangeSnapshot
		prevErr := p.db.Where("monitor_id = ? AND id < ? AND is_baseline = ?", monitor.ID, snapshot.ID, false).
			Order("created_at DESC").
			First(&prevSnapshot).Error

		if prevErr == nil && prevSnapshot.HTMLHash == htmlHash && prevSnapshot.ChangeScore >= changeThreshold {
			// Same changed content seen before - accept as new baseline
			p.rotateBaseline(monitor.ID, &snapshot, screenshotAbsPath, htmlContent)

			heartbeat.Status = StatusUp
			heartbeat.Message = fmt.Sprintf("Change accepted as new baseline (score: %.2f) - %dms", changeScore, ping)
		} else {
			// First detection of this change - report DOWN to trigger notification
			heartbeat.Status = StatusDown
			heartbeat.Message = fmt.Sprintf("Page changed (score: %.2f, threshold: %.2f) - Image: %.2f, HTML: %.2f, Runtime: %.2f - %dms",
				changeScore, changeThreshold, imageScore, htmlScore, runtimeScore, ping)
		}
	} else {
		// Changed + no auto-accept: stay DOWN until page reverts
		heartbeat.Status = StatusDown
		heartbeat.Message = fmt.Sprintf("Page changed (score: %.2f, threshold: %.2f) - Image: %.2f, HTML: %.2f, Runtime: %.2f - %dms",
			changeScore, changeThreshold, imageScore, htmlScore, runtimeScore, ping)
	}

	return heartbeat, nil
}

// rotateBaseline marks the given snapshot as the new baseline and demotes the old one.
func (p *PageChangeMonitor) rotateBaseline(monitorID int, snapshot *PageChangeSnapshot, screenshotAbsPath, htmlContent string) {
	p.db.Model(&PageChangeSnapshot{}).
		Where("monitor_id = ? AND is_baseline = ?", monitorID, true).
		Update("is_baseline", false)

	snapshot.IsBaseline = true
	p.db.Model(snapshot).Update("is_baseline", true)

	// Save HTML alongside new baseline for future granular comparison
	os.WriteFile(screenshotAbsPath+".html", []byte(htmlContent), 0644)
}

// capturePage uses Chrome to navigate to a URL and capture screenshot, HTML, and runtime metrics.
func (p *PageChangeMonitor) capturePage(ctx context.Context, url string, waitTimeMs, viewportWidth, viewportHeight int, ignoreSelectors, watchSelectors []string, customJS string) (screenshot []byte, html string, runtimeMetrics map[string]interface{}, err error) {
	// Set viewport
	if err := chromedp.Run(ctx, chromedp.EmulateViewport(int64(viewportWidth), int64(viewportHeight))); err != nil {
		return nil, "", nil, fmt.Errorf("failed to set viewport: %w", err)
	}

	// Navigate to page
	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		return nil, "", nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// Wait for page to stabilize
	if waitTimeMs > 0 {
		if err := chromedp.Run(ctx, chromedp.Sleep(time.Duration(waitTimeMs)*time.Millisecond)); err != nil {
			return nil, "", nil, fmt.Errorf("wait interrupted: %w", err)
		}
	}

	// Hide ignored elements before screenshot
	if len(ignoreSelectors) > 0 {
		hideJS := buildHideElementsJS(ignoreSelectors)
		if err := chromedp.Run(ctx, chromedp.Evaluate(hideJS, nil)); err != nil {
			log.Printf("Failed to hide ignore selectors: %v", err)
		}
	}

	// Take full-page screenshot
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&screenshot, 90)); err != nil {
		return nil, "", nil, fmt.Errorf("failed to take screenshot: %w", err)
	}

	// Restore hidden elements after screenshot
	if len(ignoreSelectors) > 0 {
		restoreJS := buildRestoreElementsJS(ignoreSelectors)
		chromedp.Run(ctx, chromedp.Evaluate(restoreJS, nil))
	}

	// Capture rendered HTML
	if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery)); err != nil {
		return nil, "", nil, fmt.Errorf("failed to capture HTML: %w", err)
	}

	// Execute runtime analysis - custom JS is user-provided code intended to run
	// on their own monitored pages (same as browser console execution)
	runtimeJS := buildRuntimeAnalysisJS(watchSelectors, customJS)
	var runtimeResult string
	if err := chromedp.Run(ctx, chromedp.Evaluate(runtimeJS, &runtimeResult)); err != nil {
		log.Printf("Runtime analysis failed: %v", err)
		runtimeMetrics = map[string]interface{}{}
	} else {
		runtimeMetrics = map[string]interface{}{}
		json.Unmarshal([]byte(runtimeResult), &runtimeMetrics)
	}

	return screenshot, html, runtimeMetrics, nil
}

func buildHideElementsJS(selectors []string) string {
	selectorsJSON, _ := json.Marshal(selectors)
	return fmt.Sprintf(`
		(function() {
			var selectors = %s;
			selectors.forEach(function(sel) {
				document.querySelectorAll(sel).forEach(function(el) {
					el.setAttribute('data-pchange-display', el.style.visibility);
					el.style.visibility = 'hidden';
				});
			});
		})();
	`, string(selectorsJSON))
}

func buildRestoreElementsJS(selectors []string) string {
	selectorsJSON, _ := json.Marshal(selectors)
	return fmt.Sprintf(`
		(function() {
			var selectors = %s;
			selectors.forEach(function(sel) {
				document.querySelectorAll(sel).forEach(function(el) {
					var prev = el.getAttribute('data-pchange-display');
					if (prev !== null) {
						el.style.visibility = prev;
						el.removeAttribute('data-pchange-display');
					}
				});
			});
		})();
	`, string(selectorsJSON))
}

func buildRuntimeAnalysisJS(watchSelectors []string, customJS string) string {
	selectorsJSON, _ := json.Marshal(watchSelectors)
	// customJS is safely JSON-encoded to prevent injection
	escapedCustomJS, _ := json.Marshal(customJS)

	return fmt.Sprintf(`
		(function() {
			var results = {
				dom_count: document.querySelectorAll('*').length,
				text_hash: '',
				network_requests: 0,
				selector_results: {}
			};

			// Simple text hash (djb2)
			var text = document.body ? document.body.innerText : '';
			var hash = 5381;
			for (var i = 0; i < text.length; i++) {
				hash = ((hash << 5) + hash) + text.charCodeAt(i);
				hash = hash & hash;
			}
			results.text_hash = hash.toString(16);

			// Network request count
			try {
				results.network_requests = performance.getEntriesByType('resource').length;
			} catch(e) {}

			// Check watch selectors
			var selectors = %s;
			selectors.forEach(function(sel) {
				results.selector_results[sel] = document.querySelector(sel) !== null;
			});

			// Execute user-provided custom JavaScript for runtime page analysis.
			// This runs in the context of the monitored page (sandboxed Chrome tab).
			var customCode = %s;
			if (customCode) {
				try {
					results.custom_js_result = Function(customCode)();
				} catch(e) {
					results.custom_js_error = e.message;
				}
			}

			return JSON.stringify(results);
		})();
	`, string(selectorsJSON), string(escapedCustomJS))
}
