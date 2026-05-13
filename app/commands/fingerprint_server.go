package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	FingerprintServerPort = 18080
)

// FingerprintServerService manages the fingerprint server process
type FingerprintServerService struct {
	mu         sync.Mutex
	serverCmd  *exec.Cmd
	serverAddr string
	running    bool
	browserPID int
}

// NewFingerprintServerService creates a new fingerprint server service
func NewFingerprintServerService() *FingerprintServerService {
	return &FingerprintServerService{}
}

// StartServer starts the fingerprint server process
func (s *FingerprintServerService) StartServer(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		// Verify it's actually responding
		if s.isServerResponding() {
			return nil
		}
		s.running = false
	}

	// Check if server is already running on the port
	if s.isPortOpen(FingerprintServerPort) && s.isServerResponding() {
		s.running = true
		s.serverAddr = fmt.Sprintf("localhost:%d", FingerprintServerPort)
		return nil
	}

	// Find the Python server script
	serverScript := s.findServerScript()
	if serverScript == "" {
		return fmt.Errorf("fingerprint server script not found")
	}

	fmt.Printf("Starting fingerprint server: %s\n", serverScript)

	// Start Python server - background context so it survives the call
	cmd := exec.CommandContext(context.Background(), "python3", serverScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start fingerprint server: %w", err)
	}

	s.serverCmd = cmd
	s.running = true
	s.serverAddr = fmt.Sprintf("localhost:%d", FingerprintServerPort)

	// Wait for server to be ready
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if s.isServerResponding() {
			fmt.Println("Fingerprint server ready")
			return nil
		}
	}

	// Server didn't respond in time
	if s.serverCmd.Process != nil {
		s.serverCmd.Process.Kill()
	}
	s.running = false
	return fmt.Errorf("server did not respond in time")
}

// findServerScript locates the Python fingerprint server script
func (s *FingerprintServerService) findServerScript() string {
	// Try common locations
	paths := []string{
		"cmd/fingerprint-server-py/server.py",
		"/Users/jin/github/light-finger-browser/cmd/fingerprint-server-py/server.py",
		filepath.Join(getAppDir(), "Resources", "cmd", "fingerprint-server-py", "server.py"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fallback: try to find via executable path
	execPath, _ := os.Executable()
	if execPath != "" {
		dir := filepath.Dir(execPath)
		// Check relative to app bundle Resources folder
		rel := filepath.Join(dir, "..", "Resources", "cmd", "fingerprint-server-py", "server.py")
		if _, err := os.Stat(rel); err == nil {
			return rel
		}
	}

	return ""
}

func getAppDir() string {
	execPath, _ := os.Executable()
	if execPath == "" {
		return ""
	}
	return filepath.Dir(execPath)
}

// StopServer stops the fingerprint server
func (s *FingerprintServerService) StopServer() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.serverCmd != nil && s.serverCmd.Process != nil {
		s.serverCmd.Process.Kill()
		s.serverCmd.Wait()
		s.serverCmd = nil
	}

	s.running = false
	return nil
}

// IsServerRunning returns whether the server is running
func (s *FingerprintServerService) IsServerRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return false
	}

	// Double-check via HTTP
	return s.isServerResponding()
}

// GetServerURL returns the server URL
func (s *FingerprintServerService) GetServerURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("http://%s", s.serverAddr)
}

// isServerResponding checks if the server responds to HTTP
func (s *FingerprintServerService) isServerResponding() bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", FingerprintServerPort))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// ==================== Browser Launch ====================

// LaunchBrowser launches Chromium with the fingerprint test page
func (s *FingerprintServerService) LaunchBrowser(ctx context.Context, browserBinaryPath string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if browserBinaryPath == "" {
		browserBinaryPath = s.detectBrowserBinary()
	}

	testURL := fmt.Sprintf("http://localhost:%d/", FingerprintServerPort)

	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{
			"--new-window",
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-dev-shm-usage",
			"--disable-extensions",
			"--disable-sync",
			"--disable-translate",
			"--remote-debugging-port=0",
			testURL,
		}
	default:
		args = []string{
			"--new-window",
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-dev-shm-usage",
			"--disable-extensions",
			"--disable-sync",
			"--disable-translate",
			"--remote-debugging-port=0",
			testURL,
		}
	}

	cmd := exec.CommandContext(context.Background(), browserBinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to launch browser: %w", err)
	}

	s.browserPID = cmd.Process.Pid
	return cmd.Process.Pid, nil
}

// CollectFingerprint collects fingerprint from the server
func (s *FingerprintServerService) CollectFingerprint(ctx context.Context) (*FingerprintVerificationResult, error) {
	serverURL := fmt.Sprintf("http://localhost:%d/api/fingerprints", FingerprintServerPort)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get fingerprints from server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var storedFingerprints map[string]StoredFingerprint
	if err := json.Unmarshal(body, &storedFingerprints); err != nil {
		return nil, fmt.Errorf("failed to parse fingerprints: %w", err)
	}

	if len(storedFingerprints) == 0 {
		return &FingerprintVerificationResult{
			Success:      false,
			ErrorMessage: "No fingerprint data collected yet",
			OverallScore: 0,
		}, nil
	}

	var latestFP *FingerprintData
	for _, fp := range storedFingerprints {
		latestFP = &fp.Data
		break
	}

	if latestFP == nil {
		return &FingerprintVerificationResult{
			Success:      false,
			ErrorMessage: "No fingerprint data available",
			OverallScore: 0,
		}, nil
	}

	result := s.calculateVerification(latestFP)
	return result, nil
}

// ==================== Similarity Algorithms ====================

// StoredFingerprint represents a stored fingerprint
type StoredFingerprint struct {
	Data      FingerprintData `json:"data"`
	CreatedAt float64        `json:"created_at"`
}

// FingerprintData represents fingerprint data
type FingerprintData struct {
	Canvas              string   `json:"canvas"`
	WebGLVendor         string   `json:"webgl_vendor"`
	WebGLRenderer       string   `json:"webgl_renderer"`
	WebGLExtensions     []string `json:"webgl_extensions"`
	AudioHash           string   `json:"audio_hash"`
	Fonts               []string `json:"fonts"`
	ScreenWidth         int      `json:"screen_width"`
	ScreenHeight        int      `json:"screen_height"`
	ScreenColorDepth    int      `json:"screen_color_depth"`
	ScreenPixelRatio    float64  `json:"screen_pixel_ratio"`
	TimezoneOffset      int      `json:"timezone_offset"`
	TimezoneName        string   `json:"timezone_name"`
	Languages           []string `json:"languages"`
	MathResults         []float64 `json:"math_results"`
	TouchMaxPoints      int      `json:"touch_max_points"`
	Platform            string   `json:"platform"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	DeviceMemory        float64  `json:"device_memory"`
	UserAgent           string   `json:"user_agent"`
	Timestamp           string   `json:"timestamp"`
}

// FingerprintVerificationResult represents verification result
type FingerprintVerificationResult struct {
	Success      bool                    `json:"success"`
	ErrorMessage string                  `json:"error_message,omitempty"`
	OverallScore float64                 `json:"overall_score"`
	FieldResults []FingerprintFieldResult `json:"field_results"`
}

// FingerprintFieldResult represents verification result for a single field
type FingerprintFieldResult struct {
	Field          string  `json:"field"`
	ExpectedValue  string  `json:"expected_value"`
	ActualValue    string  `json:"actual_value"`
	MatchScore     float64 `json:"match_score"`
	QualityRating  string  `json:"quality_rating"`
}

// FingerprintServerStatus represents the status of the fingerprint server
type FingerprintServerStatus struct {
	Running bool   `json:"running"`
	URL     string `json:"url"`
	PID     int    `json:"pid,omitempty"`
}

// GetStatus returns the current status of the fingerprint server
func (s *FingerprintServerService) GetStatus() FingerprintServerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return FingerprintServerStatus{
		Running: s.running && s.isServerResponding(),
		URL:     fmt.Sprintf("http://localhost:%d/", FingerprintServerPort),
	}
}

// Weights for fingerprint fields
var fingerprintWeights = map[string]float64{
	"canvas":               20,
	"webgl":                18,
	"audio":                15,
	"fonts":                12,
	"screen":               8,
	"timezone":             5,
	"languages":            5,
	"math":                 8,
	"touch_support":        3,
	"platform":             3,
	"hardware_concurrency": 2,
	"device_memory":        3,
}

func (s *FingerprintServerService) calculateVerification(actual *FingerprintData) *FingerprintVerificationResult {
	expected := s.getDefaultExpectedFingerprint()

	results := make([]FingerprintFieldResult, 0)
	totalWeight := 0.0
	weightedScore := 0.0

	// Canvas
	canvasScore := s.calculateStringSimilarity(expected.Canvas, actual.Canvas)
	results = append(results, FingerprintFieldResult{
		Field:         "canvas",
		ExpectedValue: truncateString(expected.Canvas, 100),
		ActualValue:   truncateString(actual.Canvas, 100),
		MatchScore:    canvasScore,
		QualityRating: getQualityRating(canvasScore),
	})
	weightedScore += canvasScore * fingerprintWeights["canvas"]
	totalWeight += fingerprintWeights["canvas"]

	// WebGL
	webglScore := (s.calculateStringSimilarity(expected.WebGLVendor, actual.WebGLVendor) +
		s.calculateStringSimilarity(expected.WebGLRenderer, actual.WebGLRenderer)) / 2
	results = append(results, FingerprintFieldResult{
		Field:         "webgl",
		ExpectedValue: fmt.Sprintf("%s / %s", expected.WebGLVendor, expected.WebGLRenderer),
		ActualValue:   fmt.Sprintf("%s / %s", actual.WebGLVendor, actual.WebGLRenderer),
		MatchScore:    webglScore,
		QualityRating: getQualityRating(webglScore),
	})
	weightedScore += webglScore * fingerprintWeights["webgl"]
	totalWeight += fingerprintWeights["webgl"]

	// Audio
	audioScore := s.calculateStringSimilarity(expected.AudioHash, actual.AudioHash)
	results = append(results, FingerprintFieldResult{
		Field:         "audio",
		ExpectedValue: expected.AudioHash,
		ActualValue:   actual.AudioHash,
		MatchScore:    audioScore,
		QualityRating: getQualityRating(audioScore),
	})
	weightedScore += audioScore * fingerprintWeights["audio"]
	totalWeight += fingerprintWeights["audio"]

	// Fonts
	fontsScore := s.calculateJaccardSimilarity(expected.Fonts, actual.Fonts)
	results = append(results, FingerprintFieldResult{
		Field:         "fonts",
		ExpectedValue: fmt.Sprintf("%d fonts", len(expected.Fonts)),
		ActualValue:   fmt.Sprintf("%d fonts", len(actual.Fonts)),
		MatchScore:    fontsScore,
		QualityRating: getQualityRating(fontsScore),
	})
	weightedScore += fontsScore * fingerprintWeights["fonts"]
	totalWeight += fingerprintWeights["fonts"]

	// Screen
	screenScore := 0.0
	if expected.ScreenWidth == actual.ScreenWidth && expected.ScreenHeight == actual.ScreenHeight {
		screenScore = 100
	}
	results = append(results, FingerprintFieldResult{
		Field:         "screen",
		ExpectedValue: fmt.Sprintf("%dx%d", expected.ScreenWidth, expected.ScreenHeight),
		ActualValue:   fmt.Sprintf("%dx%d", actual.ScreenWidth, actual.ScreenHeight),
		MatchScore:    screenScore,
		QualityRating: getQualityRating(screenScore),
	})
	weightedScore += screenScore * fingerprintWeights["screen"]
	totalWeight += fingerprintWeights["screen"]

	// Timezone
	timezoneScore := 0.0
	if expected.TimezoneOffset == actual.TimezoneOffset {
		timezoneScore = 100
	}
	results = append(results, FingerprintFieldResult{
		Field:         "timezone",
		ExpectedValue: fmt.Sprintf("UTC%s%d (%s)", formatOffset(expected.TimezoneOffset), expected.TimezoneOffset, expected.TimezoneName),
		ActualValue:   fmt.Sprintf("UTC%s%d (%s)", formatOffset(actual.TimezoneOffset), actual.TimezoneOffset, actual.TimezoneName),
		MatchScore:    timezoneScore,
		QualityRating: getQualityRating(timezoneScore),
	})
	weightedScore += timezoneScore * fingerprintWeights["timezone"]
	totalWeight += fingerprintWeights["timezone"]

	// Languages
	langsScore := s.calculateJaccardStringSimilarity(expected.Languages, actual.Languages)
	results = append(results, FingerprintFieldResult{
		Field:         "languages",
		ExpectedValue: strings.Join(expected.Languages, ", "),
		ActualValue:   strings.Join(actual.Languages, ", "),
		MatchScore:    langsScore,
		QualityRating: getQualityRating(langsScore),
	})
	weightedScore += langsScore * fingerprintWeights["languages"]
	totalWeight += fingerprintWeights["languages"]

	// Math
	mathScore := s.calculateMathSimilarity(expected.MathResults, actual.MathResults)
	results = append(results, FingerprintFieldResult{
		Field:         "math",
		ExpectedValue: "16-function vector",
		ActualValue:   "16-function vector",
		MatchScore:    mathScore,
		QualityRating: getQualityRating(mathScore),
	})
	weightedScore += mathScore * fingerprintWeights["math"]
	totalWeight += fingerprintWeights["math"]

	// Touch
	touchScore := 0.0
	if expected.TouchMaxPoints == actual.TouchMaxPoints {
		touchScore = 100
	}
	results = append(results, FingerprintFieldResult{
		Field:         "touch_support",
		ExpectedValue: fmt.Sprintf("%d points", expected.TouchMaxPoints),
		ActualValue:   fmt.Sprintf("%d points", actual.TouchMaxPoints),
		MatchScore:    touchScore,
		QualityRating: getQualityRating(touchScore),
	})
	weightedScore += touchScore * fingerprintWeights["touch_support"]
	totalWeight += fingerprintWeights["touch_support"]

	// Platform
	platformScore := s.calculateStringSimilarity(expected.Platform, actual.Platform)
	results = append(results, FingerprintFieldResult{
		Field:         "platform",
		ExpectedValue: expected.Platform,
		ActualValue:   actual.Platform,
		MatchScore:    platformScore,
		QualityRating: getQualityRating(platformScore),
	})
	weightedScore += platformScore * fingerprintWeights["platform"]
	totalWeight += fingerprintWeights["platform"]

	// Hardware concurrency
	hwScore := s.calculateNumericSimilarity(float64(expected.HardwareConcurrency), float64(actual.HardwareConcurrency), 0.15)
	results = append(results, FingerprintFieldResult{
		Field:         "hardware_concurrency",
		ExpectedValue: fmt.Sprintf("%d cores", expected.HardwareConcurrency),
		ActualValue:   fmt.Sprintf("%d cores", actual.HardwareConcurrency),
		MatchScore:    hwScore,
		QualityRating: getQualityRating(hwScore),
	})
	weightedScore += hwScore * fingerprintWeights["hardware_concurrency"]
	totalWeight += fingerprintWeights["hardware_concurrency"]

	// Device memory
	memScore := s.calculateNumericSimilarity(expected.DeviceMemory, actual.DeviceMemory, 0.15)
	results = append(results, FingerprintFieldResult{
		Field:         "device_memory",
		ExpectedValue: fmt.Sprintf("%.1f GB", expected.DeviceMemory),
		ActualValue:   fmt.Sprintf("%.1f GB", actual.DeviceMemory),
		MatchScore:    memScore,
		QualityRating: getQualityRating(memScore),
	})
	weightedScore += memScore * fingerprintWeights["device_memory"]
	totalWeight += fingerprintWeights["device_memory"]

	overallScore := 0.0
	if totalWeight > 0 {
		overallScore = weightedScore / totalWeight
	}

	return &FingerprintVerificationResult{
		Success:      overallScore >= 70,
		OverallScore: overallScore,
		FieldResults: results,
	}
}

func (s *FingerprintServerService) getDefaultExpectedFingerprint() *FingerprintData {
	return &FingerprintData{
		Canvas:              "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAMgAAADICAYAAACtWK6eAAAAAXNSR0IArs4c6QAA",
		WebGLVendor:        "Google Inc. (Apple)",
		WebGLRenderer:      "ANGLE (Apple, Vulkan 1.3.0 (Core Profile) Mesa 24.0.5)",
		WebGLExtensions:    []string{"WEBGL_debug_renderer_info"},
		AudioHash:          "audio_1234567",
		Fonts:              []string{"Arial", "Helvetica", "Times New Roman"},
		ScreenWidth:        1920,
		ScreenHeight:       1080,
		ScreenColorDepth:   24,
		ScreenPixelRatio:   1.0,
		TimezoneOffset:     -480,
		TimezoneName:       "America/Los_Angeles",
		Languages:          []string{"en-US", "en"},
		MathResults:        []float64{1, 0.523, 0.5, 0.785, 3, 2, 1, 2.718, 1, 1024, 2, 0.841, 0, 1.175, 4, 1},
		TouchMaxPoints:     0,
		Platform:           "MacIntel",
		HardwareConcurrency: 8,
		DeviceMemory:       8,
		UserAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
	}
}

func (s *FingerprintServerService) calculateStringSimilarity(a, b string) float64 {
	if a == b {
		return 100
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	distance := levenshteinDistance(a, b)
	maxLen := math.Max(float64(len(a)), float64(len(b)))
	similarity := (1 - float64(distance)/maxLen) * 100
	return math.Max(0, similarity)
}

func (s *FingerprintServerService) calculateJaccardSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 100
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setA := make(map[string]bool)
	for _, v := range a {
		setA[v] = true
	}
	setB := make(map[string]bool)
	for _, v := range b {
		setB[v] = true
	}
	intersection := 0
	for v := range setA {
		if setB[v] {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union) * 100
}

func (s *FingerprintServerService) calculateJaccardStringSimilarity(a, b []string) float64 {
	return s.calculateJaccardSimilarity(a, b)
}

func (s *FingerprintServerService) calculateNumericSimilarity(a, b, tolerance float64) float64 {
	if a == b {
		return 100
	}
	if a == 0 && b == 0 {
		return 100
	}
	diff := math.Abs(a - b)
	maxVal := math.Max(math.Abs(a), math.Abs(b))
	if maxVal == 0 {
		return 100
	}
	relativeDiff := diff / maxVal
	if relativeDiff <= tolerance {
		return 100 - relativeDiff/tolerance*50
	}
	return math.Max(0, 100-relativeDiff*100)
}

func (s *FingerprintServerService) calculateMathSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	if len(a) == 0 {
		return 100
	}
	totalDiff := 0.0
	for i := range a {
		diff := math.Abs(a[i] - b[i])
		maxVal := math.Max(math.Abs(a[i]), math.Abs(b[i]))
		if maxVal > 0 {
			totalDiff += diff / maxVal
		}
	}
	avgDiff := totalDiff / float64(len(a))
	if avgDiff < 0.01 {
		return 100
	}
	return math.Max(0, 100-avgDiff*100)
}

func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				matrix[i][j] = matrix[i-1][j-1]
			} else {
				min := matrix[i-1][j-1] + 1
				if matrix[i-1][j]+1 < min {
					min = matrix[i-1][j] + 1
				}
				if matrix[i][j-1]+1 < min {
					min = matrix[i][j-1] + 1
				}
				matrix[i][j] = min
			}
		}
	}
	return matrix[len(a)][len(b)]
}

func getQualityRating(score float64) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 70:
		return "Good"
	case score >= 50:
		return "Poor"
	default:
		return "Fail"
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func formatOffset(minutes int) string {
	if minutes == 0 {
		return "+0"
	}
	sign := "+"
	if minutes < 0 {
		sign = "-"
		minutes = -minutes
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%s%d", sign, hours)
	}
	return fmt.Sprintf("%s%d:%02d", sign, hours, mins)
}

// Helper methods
func (s *FingerprintServerService) isPortOpen(port int) bool {
	conn, err := (&net.Dialer{Timeout: 100 * time.Millisecond}).Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *FingerprintServerService) detectBrowserBinary() string {
	execPath, _ := os.Executable()
	if execPath != "" {
		dir := filepath.Dir(execPath)
		switch runtime.GOOS {
		case "darwin":
			paths := []string{
				filepath.Join(dir, "Resources", "selfbuilt", "Chromium.app", "Contents", "MacOS", "Chromium"),
				filepath.Join(dir, "selfbuilt", "Chromium.app", "Contents", "MacOS", "Chromium"),
			}
			for _, p := range paths {
				if _, err := os.Stat(p); err == nil {
					return p
				}
			}
		case "windows":
			p := filepath.Join(dir, "selfbuilt", "selfbuilt.exe")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		default:
			p := filepath.Join(dir, "selfbuilt", "selfbuilt")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	switch runtime.GOOS {
	case "darwin":
		paths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	case "windows":
		paths := []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	default:
		paths := []string{
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// GetFingerprintTestURL returns the URL for the fingerprint test page
func (s *FingerprintServerService) GetFingerprintTestURL() string {
	return fmt.Sprintf("http://localhost:%d/", FingerprintServerPort)
}

// LaunchBrowserWithChrome is deprecated
func (s *FingerprintServerService) LaunchBrowserWithChrome(ctx context.Context, instanceID string) (int, error) {
	return s.LaunchBrowser(ctx, "")
}

// KillBrowser kills the browser process
func (s *FingerprintServerService) KillBrowser(pid int) error {
	if pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
