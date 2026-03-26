package monitor

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// compareImages compares two PNG screenshots and returns a difference score (0.0-1.0)
// and a diff image highlighting changed pixels in red.
func compareImages(baseline, current []byte) (float64, []byte, error) {
	baseImg, _, err := image.Decode(bytes.NewReader(baseline))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to decode baseline image: %w", err)
	}

	currImg, _, err := image.Decode(bytes.NewReader(current))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to decode current image: %w", err)
	}

	baseBounds := baseImg.Bounds()
	currBounds := currImg.Bounds()

	// Use the larger dimensions for comparison
	maxWidth := baseBounds.Dx()
	if currBounds.Dx() > maxWidth {
		maxWidth = currBounds.Dx()
	}
	maxHeight := baseBounds.Dy()
	if currBounds.Dy() > maxHeight {
		maxHeight = currBounds.Dy()
	}

	// If dimensions differ significantly, that's a big change
	if baseBounds.Dx() != currBounds.Dx() || baseBounds.Dy() != currBounds.Dy() {
		// Dimension change contributes to score
		widthRatio := math.Abs(float64(baseBounds.Dx()-currBounds.Dx())) / float64(maxWidth)
		heightRatio := math.Abs(float64(baseBounds.Dy()-currBounds.Dy())) / float64(maxHeight)
		dimScore := (widthRatio + heightRatio) / 2

		// Still do pixel comparison on overlapping region
		overlapW := min(baseBounds.Dx(), currBounds.Dx())
		overlapH := min(baseBounds.Dy(), currBounds.Dy())

		diffImg := image.NewRGBA(image.Rect(0, 0, maxWidth, maxHeight))
		diffPixels := 0
		totalPixels := maxWidth * maxHeight

		for y := 0; y < maxHeight; y++ {
			for x := 0; x < maxWidth; x++ {
				if x < overlapW && y < overlapH {
					if pixelsDiffer(baseImg.At(baseBounds.Min.X+x, baseBounds.Min.Y+y),
						currImg.At(currBounds.Min.X+x, currBounds.Min.Y+y)) {
						diffPixels++
						diffImg.Set(x, y, color.RGBA{R: 255, A: 200})
					} else {
						r, g, b, a := currImg.At(currBounds.Min.X+x, currBounds.Min.Y+y).RGBA()
						diffImg.Set(x, y, color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)})
					}
				} else {
					diffPixels++
					diffImg.Set(x, y, color.RGBA{R: 255, G: 100, A: 200})
				}
			}
		}

		pixelScore := float64(diffPixels) / float64(totalPixels)
		score := math.Max(pixelScore, dimScore)
		if score > 1.0 {
			score = 1.0
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, diffImg); err != nil {
			return score, nil, nil
		}
		return score, buf.Bytes(), nil
	}

	// Same dimensions - standard pixel comparison
	diffImg := image.NewRGBA(image.Rect(0, 0, maxWidth, maxHeight))
	diffPixels := 0
	totalPixels := maxWidth * maxHeight

	for y := 0; y < maxHeight; y++ {
		for x := 0; x < maxWidth; x++ {
			bx := baseBounds.Min.X + x
			by := baseBounds.Min.Y + y
			cx := currBounds.Min.X + x
			cy := currBounds.Min.Y + y

			if pixelsDiffer(baseImg.At(bx, by), currImg.At(cx, cy)) {
				diffPixels++
				diffImg.Set(x, y, color.RGBA{R: 255, A: 200})
			} else {
				r, g, b, a := currImg.At(cx, cy).RGBA()
				diffImg.Set(x, y, color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)})
			}
		}
	}

	score := float64(diffPixels) / float64(totalPixels)

	// Skip encoding diff image if no differences found
	if diffPixels == 0 {
		return 0, nil, nil
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, diffImg); err != nil {
		return score, nil, nil
	}
	return score, buf.Bytes(), nil
}

// pixelsDiffer checks if two pixels differ beyond a tolerance threshold.
func pixelsDiffer(a, b color.Color) bool {
	const tolerance = 5 * 257 // tolerance per channel (5 out of 255, scaled to 16-bit)
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return absDiff(r1, r2) > tolerance ||
		absDiff(g1, g2) > tolerance ||
		absDiff(b1, b2) > tolerance ||
		absDiff(a1, a2) > tolerance
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// compareHTML compares two HTML strings and returns a change score (0.0-1.0)
// based on line-level differences.
func compareHTML(baseline, current string) float64 {
	baseLines := strings.Split(baseline, "\n")
	currLines := strings.Split(current, "\n")

	// Build set of baseline lines
	baseSet := make(map[string]int, len(baseLines))
	for _, line := range baseLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			baseSet[trimmed]++
		}
	}

	// Count current lines not in baseline
	totalLines := 0
	changedLines := 0
	for _, line := range currLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		totalLines++
		if count, ok := baseSet[trimmed]; ok && count > 0 {
			baseSet[trimmed]--
		} else {
			changedLines++
		}
	}

	// Also count baseline lines not in current (deleted lines)
	for _, count := range baseSet {
		if count > 0 {
			changedLines += count
		}
	}

	// Use max of both line counts as denominator
	baseNonEmpty := 0
	for _, line := range baseLines {
		if strings.TrimSpace(line) != "" {
			baseNonEmpty++
		}
	}
	denominator := max(totalLines, baseNonEmpty)
	if denominator == 0 {
		return 0
	}

	score := float64(changedLines) / float64(denominator)
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// compareRuntime compares two runtime metric snapshots and returns a change score (0.0-1.0).
func compareRuntime(baseline, current map[string]interface{}) float64 {
	if baseline == nil || current == nil {
		if baseline == nil && current == nil {
			return 0
		}
		return 1.0
	}

	scores := make([]float64, 0, 4)

	// Compare DOM element count
	baseDOMCount := getFloat(baseline, "dom_count")
	currDOMCount := getFloat(current, "dom_count")
	if baseDOMCount > 0 {
		domRatio := math.Abs(currDOMCount-baseDOMCount) / baseDOMCount
		if domRatio > 1.0 {
			domRatio = 1.0
		}
		scores = append(scores, domRatio)
	}

	// Compare text content hash
	baseTextHash, _ := baseline["text_hash"].(string)
	currTextHash, _ := current["text_hash"].(string)
	if baseTextHash != "" && currTextHash != "" {
		if baseTextHash != currTextHash {
			scores = append(scores, 0.5)
		} else {
			scores = append(scores, 0)
		}
	}

	// Compare selector results
	baseSelectors := getMapStringBool(baseline, "selector_results")
	currSelectors := getMapStringBool(current, "selector_results")
	if len(baseSelectors) > 0 {
		changed := 0
		for sel, basePresent := range baseSelectors {
			if currPresent, ok := currSelectors[sel]; ok {
				if basePresent != currPresent {
					changed++
				}
			} else {
				changed++
			}
		}
		selectorScore := float64(changed) / float64(len(baseSelectors))
		scores = append(scores, selectorScore)
	}

	// Compare network request count
	baseNetCount := getFloat(baseline, "network_requests")
	currNetCount := getFloat(current, "network_requests")
	if baseNetCount > 0 {
		netRatio := math.Abs(currNetCount-baseNetCount) / baseNetCount
		if netRatio > 1.0 {
			netRatio = 1.0
		}
		scores = append(scores, netRatio*0.5) // Network changes weighted lower
	}

	if len(scores) == 0 {
		return 0
	}

	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	return sum / float64(len(scores))
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	if v, ok := m[key].(json.Number); ok {
		f, _ := v.Float64()
		return f
	}
	return 0
}

func getMapStringBool(m map[string]interface{}, key string) map[string]bool {
	result := make(map[string]bool)
	raw, ok := m[key].(map[string]interface{})
	if !ok {
		return result
	}
	for k, v := range raw {
		if b, ok := v.(bool); ok {
			result[k] = b
		}
	}
	return result
}

// ensureStorageDir creates the directory path if it doesn't exist.
func ensureStorageDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// screenshotDir returns the storage directory for a specific monitor's screenshots.
func screenshotDir(basePath string, monitorID int) string {
	return filepath.Join(basePath, fmt.Sprintf("%d", monitorID))
}

// hashString returns a hex-encoded SHA-256 hash of the input string.
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

// Config helper methods for PageChangeMonitor

func getConfigFloat(monitor *Monitor, key string, defaultValue float64) float64 {
	if monitor.Config == nil {
		return defaultValue
	}
	if val, ok := monitor.Config[key].(float64); ok {
		return val
	}
	return defaultValue
}

func getConfigString(monitor *Monitor, key, defaultValue string) string {
	if monitor.Config == nil {
		return defaultValue
	}
	if val, ok := monitor.Config[key].(string); ok {
		return val
	}
	return defaultValue
}

func getConfigInt(monitor *Monitor, key string, defaultValue int) int {
	if monitor.Config == nil {
		return defaultValue
	}
	if val, ok := monitor.Config[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func getConfigBool(monitor *Monitor, key string, defaultValue bool) bool {
	if monitor.Config == nil {
		return defaultValue
	}
	if val, ok := monitor.Config[key].(bool); ok {
		return val
	}
	return defaultValue
}

func getConfigStringSlice(monitor *Monitor, key string) []string {
	if monitor.Config == nil {
		return nil
	}
	if val, ok := monitor.Config[key].([]interface{}); ok {
		result := make([]string, 0, len(val))
		for _, v := range val {
			if s, ok := v.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	}
	// Also support comma/newline-separated strings
	if val, ok := monitor.Config[key].(string); ok && val != "" {
		lines := strings.Split(val, "\n")
		result := make([]string, 0, len(lines))
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return nil
}
