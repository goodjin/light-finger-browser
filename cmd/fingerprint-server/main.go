package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FingerprintData struct {
	Canvas             string   `json:"canvas"`
	WebGLVendor        string   `json:"webgl_vendor"`
	WebGLRenderer      string   `json:"webgl_renderer"`
	WebGLExtensions    []string `json:"webgl_extensions"`
	AudioHash          string   `json:"audio_hash"`
	Fonts              []string `json:"fonts"`
	ScreenWidth        int      `json:"screen_width"`
	ScreenHeight       int      `json:"screen_height"`
	ScreenColorDepth   int      `json:"screen_color_depth"`
	ScreenPixelRatio   float64  `json:"screen_pixel_ratio"`
	TimezoneOffset     int      `json:"timezone_offset"`
	TimezoneName       string   `json:"timezone_name"`
	Languages          []string `json:"languages"`
	MathResults        []float64 `json:"math_results"`
	TouchMaxPoints     int      `json:"touch_max_points"`
	Platform           string   `json:"platform"`
	HardwareConcurrency int     `json:"hardware_concurrency"`
	DeviceMemory       float64  `json:"device_memory"`
	UserAgent          string   `json:"user_agent"`
	Timestamp          string   `json:"timestamp"`
}

type StoredFingerprint struct {
	Data      FingerprintData `json:"data"`
	CreatedAt time.Time      `json:"created_at"`
}

var (
	mu           sync.RWMutex
	fingerprints = make(map[string]StoredFingerprint)
)

// getFingerprintHTML returns the fingerprint test HTML page with iframe injection JS
func getFingerprintHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Fingerprint Test</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      margin: 0;
      padding: 20px;
      background: #f5f5f5;
    }
    .container {
      max-width: 800px;
      margin: 0 auto;
      background: white;
      padding: 24px;
      border-radius: 8px;
      box-shadow: 0 2px 8px rgba(0,0,0,0.1);
    }
    h1 { color: #333; margin-bottom: 20px; }
    .status { padding: 12px; border-radius: 6px; margin-bottom: 20px; }
    .status.collecting { background: #fff3cd; color: #856404; }
    .status.ready { background: #d4edda; color: #155724; }
    .status.error { background: #f8d7da; color: #721c24; }
    .result { margin-top: 20px; }
    .result pre {
      background: #1a1a2e;
      color: #e5e7eb;
      padding: 16px;
      border-radius: 6px;
      overflow-x: auto;
      font-size: 12px;
    }
    iframe { display: none; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Fingerprint Test Page</h1>
    <div id="status" class="status collecting">Collecting fingerprint data...</div>
    <iframe id="fingerprint-iframe"></iframe>
    <div id="result" class="result" style="display:none;">
      <h3>Collected Fingerprint:</h3>
      <pre id="fingerprint-data"></pre>
    </div>
  </div>

  <script>
    (function() {
      // Collect fingerprint data
      function collectFingerprint() {
        var data = {
          canvas: collectCanvas(),
          webgl_vendor: collectWebGLVendor(),
          webgl_renderer: collectWebGLRenderer(),
          webgl_extensions: collectWebGLExtensions(),
          audio_hash: collectAudioHash(),
          fonts: collectFonts(),
          screen_width: screen.width,
          screen_height: screen.height,
          screen_color_depth: screen.colorDepth,
          screen_pixel_ratio: window.devicePixelRatio || 1,
          timezone_offset: new Date().getTimezoneOffset(),
          timezone_name: Intl.DateTimeFormat().resolvedOptions().timeZone || '',
          languages: collectLanguages(),
          math_results: collectMathResults(),
          touch_max_points: navigator.maxTouchPoints || 0,
          platform: navigator.platform || '',
          hardware_concurrency: navigator.hardwareConcurrency || 0,
          device_memory: navigator.deviceMemory || 0,
          user_agent: navigator.userAgent || '',
          timestamp: new Date().toISOString()
        };
        return data;
      }

      function collectCanvas() {
        try {
          var canvas = document.createElement('canvas');
          canvas.width = 200;
          canvas.height = 50;
          var ctx = canvas.getContext('2d');
          // Text rendering
          ctx.textBaseline = 'top';
          ctx.font = '14px Arial';
          ctx.fillStyle = '#f60';
          ctx.fillRect(125, 1, 62, 20);
          ctx.fillStyle = '#069';
          ctx.fillText('Fingerprint Test', 2, 15);
          ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
          ctx.fillText('Fingerprint Test', 4, 17);
          // Geometry rendering
          ctx.beginPath();
          ctx.arc(50, 25, 20, 0, Math.PI * 2);
          ctx.stroke();
          return canvas.toDataURL();
        } catch (e) {
          return 'error';
        }
      }

      function collectWebGLVendor() {
        try {
          var canvas = document.createElement('canvas');
          var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
          if (!gl) return '';
          var ext = gl.getExtension('WEBGL_debug_renderer_info');
          if (!ext) return '';
          return gl.getParameter(ext.UNMASKED_VENDOR_WEBGL) || '';
        } catch (e) {
          return '';
        }
      }

      function collectWebGLRenderer() {
        try {
          var canvas = document.createElement('canvas');
          var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
          if (!gl) return '';
          var ext = gl.getExtension('WEBGL_debug_renderer_info');
          if (!ext) return '';
          return gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) || '';
        } catch (e) {
          return '';
        }
      }

      function collectWebGLExtensions() {
        try {
          var canvas = document.createElement('canvas');
          var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
          if (!gl) return [];
          return gl.getSupportedExtensions() || [];
        } catch (e) {
          return [];
        }
      }

      function collectAudioHash() {
        try {
          var AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
          if (!AudioContext) return '';
          var context = new AudioContext(1, 44100, 44100);
          var oscillator = context.createOscillator();
          var analyser = context.createAnalyser();
          var gain = context.createGain();
          var processor = context.createScriptProcessor(4096, 1, 1);

          gain.gain.value = 0;
          oscillator.type = 'triangle';
          oscillator.frequency.value = 10000;

          oscillator.connect(analyser);
          analyser.connect(processor);
          processor.connect(gain);
          gain.connect(context.destination);

          oscillator.start(0);

          var hash = 0;
          processor.onaudioprocess = function(event) {
            var data = event.inputBuffer.getChannelData(0);
            for (var i = 0; i < data.length; i++) {
              hash = (hash << 5) - hash + data[i];
              hash = hash & hash;
            }
          };

          // Give it some time then stop
          setTimeout(function() {
            context.close();
          }, 100);

          return 'audio_' + Math.abs(hash);
        } catch (e) {
          return '';
        }
      }

      function collectFonts() {
        var baseFonts = ['monospace', 'sans-serif', 'serif'];
        var testString = 'mmmmmmmmmmlli';
        var testSize = '72px';
        var canvas = document.createElement('canvas');
        var ctx = canvas.getContext('2d');

        function getWidth(font) {
          ctx.font = testSize + ' ' + font;
          return ctx.measureText(testString).width;
        }

        function detectFont(font) {
          var detected = false;
          for (var i = 0; i < baseFonts.length; i++) {
            var baseWidth = getWidth(baseFonts[i]);
            var testWidth = getWidth(font + ',' + baseFonts[i]);
            if (baseWidth !== testWidth) {
              detected = true;
              break;
            }
          }
          return detected;
        }

        var commonFonts = [
          'Arial', 'Arial Black', 'Arial Narrow', 'Calibri', 'Cambria',
          'Comic Sans MS', 'Consolas', 'Courier', 'Courier New', 'Georgia',
          'Helvetica', 'Impact', 'Lucida Console', 'Lucida Sans Unicode',
          'Microsoft Sans Serif', 'Monaco', 'Palatino Linotype', 'Segoe UI',
          'Tahoma', 'Times', 'Times New Roman', 'Trebuchet MS', 'Verdana',
          'Helvetica Neue', 'SF Pro Display', 'SF Pro Text', 'Menlo'
        ];

        var detectedFonts = [];
        for (var i = 0; i < commonFonts.length; i++) {
          if (detectFont(commonFonts[i])) {
            detectedFonts.push(commonFonts[i]);
          }
        }
        return detectedFonts;
      }

      function collectLanguages() {
        var langs = [];
        if (navigator.languages && navigator.languages.length) {
          langs = Array.prototype.slice.call(navigator.languages);
        } else if (navigator.language) {
          langs = [navigator.language];
        }
        return langs;
      }

      function collectMathResults() {
        var results = [];
        // Test 22 math functions that can differentiate browsers
        var tests = [
          ['abs', [-1]], ['acos', [0.5]], ['acosh', [1.5]], ['asin', [0.5]],
          ['asinh', [1.5]], ['atan', [1]], ['atanh', [0.5]], ['atan2', [1, 1]],
          ['cbrt', [27]], ['ceil', [1.5]], ['cos', [0]], ['cosh', [1]],
          ['exp', [1]], ['expm1', [1]], ['floor', [1.5]], ['fround', [1.337]],
          ['log', [Math.E]], ['log1p', [Math.E - 1]], ['log10', [100]],
          ['log2', [1024]], ['pow', [2, 10]], ['round', [1.5]],
          ['sign', [-5]], ['sin', [0]], ['sinh', [1]], ['sqrt', [16]],
          ['tan', [0]], ['tanh', [1]], ['trunc', [1.5]],
          ['clz32', [255]], ['imul', [5, 7]], ['max', [1,2,3]], ['min', [1,2,3]],
          ['hypot', [3,4]]
        ];

        for (var i = 0; i < tests.length && i < 22; i++) {
          try {
            var fn = Math[tests[i][0]];
            if (typeof fn === 'function') {
              var result = fn.apply(Math, tests[i][1]);
              results.push(parseFloat(result.toPrecision(12)));
            }
          } catch (e) {
            results.push(0);
          }
        }
        return results;
      }

      // Send data to parent via postMessage
      function sendToParent(data) {
        if (window.parent !== window) {
          window.parent.postMessage({ type: 'fingerprint-result', data: data }, '*');
        }
      }

      // Load iframe with srcdoc to avoid CORS
      function initIframe() {
        var iframe = document.getElementById('fingerprint-iframe');
        // Create srcdoc content that will collect fingerprint and send to parent
        var iframeContent = '<!DOCTYPE html><html><head></head><body>' +
          '<script>' +
          '(function() {' +
          '  var data = parent.collectFingerprint ? parent.collectFingerprint() : {};' +
          '  if (window.parent !== window) {' +
          '    window.parent.postMessage({ type: ' + "'fingerprint-result'" + ', data: data }, ' + "'*'" + ');' +
          '  }' +
          '})();' +
          '</' + 'script></body></html>';

        // Wait for parent to be ready
        setTimeout(function() {
          var data = collectFingerprint();
          document.getElementById('fingerprint-data').textContent = JSON.stringify(data, null, 2);
          document.getElementById('status').className = 'status ready';
          document.getElementById('status').textContent = 'Fingerprint collected successfully';
          document.getElementById('result').style.display = 'block';

          // Send to parent
          sendToParent(data);

          // Also POST to server
          fetch('/api/fingerprint', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
          }).catch(function(err) {
            console.log('Failed to send to server:', err);
          });
        }, 500);
      }

      // Start collection when page loads
      if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initIframe);
      } else {
        initIframe();
      }

      // Expose collectFingerprint for iframe use
      window.collectFingerprint = collectFingerprint;
    })();
  </script>
</body>
</html>`
}

func handleFingerprint(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	// Handle preflight OPTIONS request
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(getFingerprintHTML()))
		return
	}

	if r.Method == http.MethodPost && r.URL.Path == "/api/fingerprint" {
		var data FingerprintData
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		mu.Lock()
		fingerprints[data.UserAgent] = StoredFingerprint{
			Data:      data,
			CreatedAt: time.Now(),
		}
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.NotFound(w, r)
}

func handleGetFingerprints(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string]StoredFingerprint)
	for k, v := range fingerprints {
		result[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// setCORSHeaders sets CORS headers for cross-origin requests from the frontend
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func main() {
	port := 18080
	if envPort := strings.TrimSpace(os.Getenv("FINGERPRINT_SERVER_PORT")); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleFingerprint)
	mux.HandleFunc("/api/fingerprint", handleFingerprint)
	mux.HandleFunc("/api/fingerprints", handleGetFingerprints)
	mux.HandleFunc("/health", handleHealth)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Fingerprint server starting on %s", addr)
	log.Printf("Test page: http://localhost:%d/", port)
	log.Printf("API endpoint: http://localhost:%d/api/fingerprint", port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
