package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type FingerprintSnapshot struct {
	UserAgent  string            `json:"user_agent"`
	Platform   string            `json:"platform"`
	Language   string            `json:"language"`
	Languages  []string          `json:"languages"`
	Timezone   string            `json:"timezone"`
	Screen     FingerprintScreen `json:"screen"`
	WebGL      FingerprintWebGL  `json:"webgl"`
	CanvasHash string            `json:"canvas_hash"`
	AudioHash  string            `json:"audio_hash"`
}

type FingerprintScreen struct {
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	PixelRatio float64 `json:"pixel_ratio"`
	ColorDepth int     `json:"color_depth"`
}

type FingerprintWebGL struct {
	Vendor   string `json:"vendor"`
	Renderer string `json:"renderer"`
}

type FingerprintCheckResult struct {
	Snapshot     *FingerprintSnapshot `json:"snapshot"`
	Expected     *FingerprintSnapshot `json:"expected"`
	Previous     *FingerprintSnapshot `json:"previous"`
	Matches      bool                 `json:"matches"`
	Diffs        []string             `json:"diffs"`
	CoverageGaps []string             `json:"coverage_gaps"`
	Timestamp    string               `json:"timestamp"`
}

func (s *AccountService) CheckFingerprint(ctx context.Context, instanceID string) (*FingerprintCheckResult, error) {
	inst, err := s.instanceStore.Get(instanceID)
	if err != nil {
		return nil, err
	}

	wsURL, err := resolveWebSocketURL(inst.Port)
	if err != nil {
		return nil, err
	}

	snapshot, err := collectFingerprint(ctx, wsURL)
	if err != nil {
		return nil, err
	}

	expected := expectedFingerprintSnapshot(inst.Fingerprint)
	var previousSnapshot *FingerprintSnapshot

	if inst.AccountID != "" {
		prev, err := s.snapshotStore.GetLatestByAccount(inst.AccountID)
		if err == nil && prev != nil && prev.Snapshot != "" {
			_ = json.Unmarshal([]byte(prev.Snapshot), &previousSnapshot)
		}
	}

	diffs := diffFingerprint(expected, snapshot)
	matches := len(diffs) == 0
	coverageGaps := currentVerificationCoverageGaps()

	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}

	if _, err := s.snapshotStore.Save(&sqlite.FingerprintSnapshot{
		ID:         uuid.New().String(),
		AccountID:  inst.AccountID,
		InstanceID: inst.ID,
		ProxyID:    inst.ProxyID,
		Snapshot:   string(snapshotJSON),
	}); err != nil {
		return nil, err
	}

	return &FingerprintCheckResult{
		Snapshot:     snapshot,
		Expected:     expected,
		Previous:     previousSnapshot,
		Matches:      matches,
		Diffs:        diffs,
		CoverageGaps: coverageGaps,
		Timestamp:    time.Now().Format(time.RFC3339Nano),
	}, nil
}

type cdpTarget struct {
	Type                 string `json:"type"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func resolveWebSocketURL(port int) (string, error) {
	url := fmt.Sprintf("http://localhost:%d/json", port)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cdp endpoint returned %d", resp.StatusCode)
	}

	var targets []cdpTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return "", err
	}
	for _, t := range targets {
		if t.Type == "page" && t.WebSocketDebuggerURL != "" {
			return t.WebSocketDebuggerURL, nil
		}
	}
	if len(targets) > 0 && targets[0].WebSocketDebuggerURL != "" {
		return targets[0].WebSocketDebuggerURL, nil
	}
	return "", fmt.Errorf("no CDP targets available")
}

func collectFingerprint(ctx context.Context, wsURL string) (*FingerprintSnapshot, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	script := `(async () => {
    function fnvHashByte(h, byte) {
      h ^= byte;
      h += (h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24);
      return h >>> 0;
    }
    function hashFloat32Array(values) {
      let h = 2166136261;
      const view = new DataView(new ArrayBuffer(4));
      for (let i = 0; i < values.length; i++) {
        view.setFloat32(0, values[i], true);
        h = fnvHashByte(h, view.getUint8(0));
        h = fnvHashByte(h, view.getUint8(1));
        h = fnvHashByte(h, view.getUint8(2));
        h = fnvHashByte(h, view.getUint8(3));
      }
      return (h >>> 0).toString(16);
    }
    function canvasHash() {
      try {
        const canvas = document.createElement('canvas');
        canvas.width = 200;
        canvas.height = 50;
        const ctx = canvas.getContext('2d');
        if (!ctx) return '';
        const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height).data;
        let h = 2166136261;
        for (let i = 0; i < imageData.length; i += 4) {
          const noise = (imageData[i] & 1) | ((imageData[i + 1] & 1) << 1) | ((imageData[i + 2] & 1) << 2);
          h = fnvHashByte(h, noise);
        }
        return (h >>> 0).toString(16);
      } catch (e) {
        return '';
      }
    }
    function webglInfo() {
      try {
        const canvas = document.createElement('canvas');
        const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return { vendor: '', renderer: '' };
        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
        if (!debugInfo) return { vendor: '', renderer: '' };
        return {
          vendor: gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL),
          renderer: gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL)
        };
      } catch (e) {
        return { vendor: '', renderer: '' };
      }
    }
    async function audioHash() {
      try {
        if (typeof AudioBuffer !== 'function') return '';
        const sampleCount = 128;
        const buffer = new AudioBuffer({ length: sampleCount, sampleRate: 44100, numberOfChannels: 1 });
        const base = new Float32Array(sampleCount);
        for (let i = 0; i < base.length; i++) {
          base[i] = (i % 10 - 5) / 10;
        }
        buffer.copyToChannel(base, 0);
        const adjusted = buffer.getChannelData(0);
        return hashFloat32Array(adjusted);
      } catch (e) {
        return '';
      }
    }
    const fp = {
      user_agent: navigator.userAgent,
      platform: navigator.platform,
      language: navigator.language,
      languages: navigator.languages || [],
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      screen: {
        width: screen.width,
        height: screen.height,
        pixel_ratio: window.devicePixelRatio || 1,
        color_depth: screen.colorDepth
      },
      webgl: webglInfo(),
      canvas_hash: canvasHash(),
      audio_hash: await audioHash()
    };
    return JSON.stringify(fp);
  })()`

	payload := map[string]interface{}{
		"id":     1,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{
			"expression":            script,
			"returnByValue":         true,
			"awaitPromise":          true,
			"userGesture":           true,
			"includeCommandLineAPI": false,
		},
	}

	if err := conn.WriteJSON(payload); err != nil {
		return nil, err
	}

	for {
		var resp map[string]interface{}
		if err := conn.ReadJSON(&resp); err != nil {
			return nil, err
		}
		if !isResponseID(resp, 1) {
			continue
		}
		if errObj, ok := resp["error"]; ok && errObj != nil {
			return nil, fmt.Errorf("cdp error: %v", errObj)
		}
		result, ok := resp["result"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected CDP response")
		}
		value := ""
		if inner, ok := result["result"].(map[string]interface{}); ok {
			if v, ok := inner["value"].(string); ok {
				value = v
			}
		}
		if value == "" {
			return nil, fmt.Errorf("empty fingerprint result")
		}
		var snapshot FingerprintSnapshot
		if err := json.Unmarshal([]byte(value), &snapshot); err != nil {
			return nil, err
		}
		return &snapshot, nil
	}
}

func diffFingerprint(prev *FingerprintSnapshot, current *FingerprintSnapshot) []string {
	if prev == nil || current == nil {
		return nil
	}
	var diffs []string
	if prev.UserAgent != "" && prev.UserAgent != current.UserAgent {
		diffs = append(diffs, "user_agent")
	}
	if prev.Platform != "" && prev.Platform != current.Platform {
		diffs = append(diffs, "platform")
	}
	if prev.Language != "" && prev.Language != current.Language {
		diffs = append(diffs, "language")
	}
	if prev.Timezone != "" && prev.Timezone != current.Timezone {
		diffs = append(diffs, "timezone")
	}
	if (prev.Screen.Width != 0 && prev.Screen.Width != current.Screen.Width) ||
		(prev.Screen.Height != 0 && prev.Screen.Height != current.Screen.Height) ||
		(prev.Screen.PixelRatio != 0 && prev.Screen.PixelRatio != current.Screen.PixelRatio) ||
		(prev.Screen.ColorDepth != 0 && prev.Screen.ColorDepth != current.Screen.ColorDepth) {
		diffs = append(diffs, "screen")
	}
	if (prev.WebGL.Vendor != "" && prev.WebGL.Vendor != current.WebGL.Vendor) ||
		(prev.WebGL.Renderer != "" && prev.WebGL.Renderer != current.WebGL.Renderer) {
		diffs = append(diffs, "webgl")
	}
	if prev.CanvasHash != "" && prev.CanvasHash != current.CanvasHash {
		diffs = append(diffs, "canvas_hash")
	}
	if prev.AudioHash != "" && prev.AudioHash != current.AudioHash {
		diffs = append(diffs, "audio_hash")
	}
	return diffs
}

func expectedFingerprintSnapshot(fp *fingerprint.Fingerprint) *FingerprintSnapshot {
	if fp == nil {
		return nil
	}
	language := strings.TrimSpace(fp.Locale)
	languages := []string{}
	if language != "" {
		languages = append(languages, language)
	}
	return &FingerprintSnapshot{
		UserAgent: fp.UserAgent,
		Platform:  fp.Platform,
		Language:  language,
		Languages: languages,
		Timezone:  fp.Timezone,
		Screen: FingerprintScreen{
			Width:      fp.Screen.Width,
			Height:     fp.Screen.Height,
			PixelRatio: fp.Screen.PixelRatio,
		},
		WebGL: FingerprintWebGL{
			Vendor:   fp.WebGL.Vendor,
			Renderer: fp.WebGL.Renderer,
		},
		CanvasHash: fp.Canvas.Hash,
		AudioHash:  fp.Audio.Hash,
	}
}

func currentVerificationCoverageGaps() []string {
	report := GetFingerprintCoverageReport()
	if report == nil {
		return nil
	}
	var gaps []string
	for _, field := range report.Fields {
		status := field.SelfBuilt
		if report.ActiveEngine == string(browserEngineLocal) {
			status = field.LocalChrome
		}
		if status == "unsupported" || status == "metadata-only" {
			gaps = append(gaps, field.Field)
		}
	}
	return gaps
}

func isResponseID(resp map[string]interface{}, expected int) bool {
	raw, ok := resp["id"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case float64:
		return int(v) == expected
	case int:
		return v == expected
	case json.Number:
		n, err := v.Int64()
		return err == nil && int(n) == expected
	default:
		return false
	}
}
