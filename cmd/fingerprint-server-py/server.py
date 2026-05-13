#!/usr/bin/env python3
"""
Fingerprint Test Server - Standalone HTTP server for browser fingerprint testing.
Can be run independently or managed by the Wails app.
"""

import json
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse

PORT = 18080

# In-memory fingerprint storage
fingerprints = {}

FINGERPRINT_HTML = '''<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Fingerprint Test</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
    .container { max-width: 900px; margin: 0 auto; background: white; padding: 24px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    h1 { color: #333; margin-bottom: 20px; }
    .status { padding: 12px; border-radius: 6px; margin-bottom: 20px; }
    .status.collecting { background: #fff3cd; color: #856404; }
    .status.ready { background: #d4edda; color: #155724; }
    .result { margin-top: 20px; }
    .result pre { background: #1a1a2e; color: #e5e7eb; padding: 16px; border-radius: 6px; overflow-x: auto; font-size: 12px; max-height: 400px; overflow-y: auto; }
    table { width: 100%; border-collapse: collapse; margin-top: 20px; }
    th, td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #ddd; }
    th { background: #f5f5f5; font-weight: 600; }
    .score-high { color: #22c55e; }
    .score-mid { color: #eab308; }
    .score-low { color: #ef4444; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Browser Fingerprint Test</h1>
    <div id="status" class="status collecting">Collecting fingerprint data...</div>
    <div id="table" style="display:none;">
      <h3>Fingerprint Components:</h3>
      <table id="fp-table">
        <thead><tr><th>Component</th><th>Value</th></tr></thead>
        <tbody id="fp-tbody"></tbody>
      </table>
    </div>
    <div id="result" class="result" style="display:none;">
      <h3>Raw Data:</h3>
      <pre id="fingerprint-data"></pre>
    </div>
  </div>
  <script>
    function collectFingerprint() {
      return {
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
        languages: navigator.languages || [navigator.language],
        math_results: collectMathResults(),
        touch_max_points: navigator.maxTouchPoints || 0,
        platform: navigator.platform || '',
        hardware_concurrency: navigator.hardwareConcurrency || 0,
        device_memory: navigator.deviceMemory || 0,
        user_agent: navigator.userAgent || '',
        timestamp: new Date().toISOString()
      };
    }

    function collectCanvas() {
      try {
        var canvas = document.createElement('canvas');
        canvas.width = 200; canvas.height = 50;
        var ctx = canvas.getContext('2d');
        ctx.textBaseline = 'top';
        ctx.font = '14px Arial';
        ctx.fillStyle = '#f60';
        ctx.fillRect(125, 1, 62, 20);
        ctx.fillStyle = '#069';
        ctx.fillText('Fingerprint Test', 2, 15);
        ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
        ctx.fillText('Fingerprint Test', 4, 17);
        ctx.beginPath();
        ctx.arc(50, 25, 20, 0, Math.PI * 2);
        ctx.stroke();
        return canvas.toDataURL().substring(0, 100) + '...';
      } catch (e) { return 'error'; }
    }

    function collectWebGLVendor() {
      try {
        var canvas = document.createElement('canvas');
        var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';
        var ext = gl.getExtension('WEBGL_debug_renderer_info');
        if (!ext) return '';
        return gl.getParameter(ext.UNMASKED_VENDOR_WEBGL) || '';
      } catch (e) { return ''; }
    }

    function collectWebGLRenderer() {
      try {
        var canvas = document.createElement('canvas');
        var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return '';
        var ext = gl.getExtension('WEBGL_debug_renderer_info');
        if (!ext) return '';
        return gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) || '';
      } catch (e) { return ''; }
    }

    function collectWebGLExtensions() {
      try {
        var canvas = document.createElement('canvas');
        var gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
        if (!gl) return [];
        return (gl.getSupportedExtensions() || []).slice(0, 10);
      } catch (e) { return []; }
    }

    function collectAudioHash() {
      try {
        var AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
        if (!AudioContext) return '';
        var context = new AudioContext(1, 44100, 44100);
        var oscillator = context.createOscillator();
        var analyser = context.createAnalyser();
        var processor = context.createScriptProcessor(4096, 1, 1);
        oscillator.connect(analyser);
        analyser.connect(processor);
        processor.connect(context.destination);
        oscillator.start(0);
        var hash = 0;
        processor.onaudioprocess = function(event) {
          var data = event.inputBuffer.getChannelData(0);
          for (var i = 0; i < data.length; i++) {
            hash = (hash << 5) - hash + data[i];
            hash = hash & hash;
          }
        };
        setTimeout(function() { context.close(); }, 100);
        return 'audio_' + Math.abs(hash);
      } catch (e) { return ''; }
    }

    function collectFonts() {
      var baseFonts = ['monospace', 'sans-serif', 'serif'];
      var testString = 'mmmmmmmmmmlli';
      var testSize = '72px';
      var canvas = document.createElement('canvas');
      var ctx = canvas.getContext('2d');
      function getWidth(font) { ctx.font = testSize + ' ' + font; return ctx.measureText(testString).width; }
      function detectFont(font) {
        for (var i = 0; i < baseFonts.length; i++) {
          var baseWidth = getWidth(baseFonts[i]);
          var testWidth = getWidth(font + ',' + baseFonts[i]);
          if (baseWidth !== testWidth) return true;
        }
        return false;
      }
      var commonFonts = ['Arial', 'Arial Black', 'Calibri', 'Consolas', 'Courier New', 'Georgia', 'Helvetica', 'Impact', 'Lucida Console', 'Monaco', 'Segoe UI', 'Tahoma', 'Times New Roman', 'Trebuchet MS', 'Verdana'];
      var detected = [];
      for (var i = 0; i < commonFonts.length; i++) {
        if (detectFont(commonFonts[i])) detected.push(commonFonts[i]);
      }
      return detected;
    }

    function collectMathResults() {
      var results = [];
      var tests = [['abs', [-1]], ['acos', [0.5]], ['asin', [0.5]], ['atan', [1]], ['atan2', [1, 1]], ['cbrt', [27]], ['ceil', [1.5]], ['cos', [0]], ['exp', [1]], ['floor', [1.5]], ['log', [Math.E]], ['pow', [2, 10]], ['round', [1.5]], ['sin', [0]], ['sqrt', [16]], ['tan', [0]]];
      for (var i = 0; i < tests.length; i++) {
        try {
          var fn = Math[tests[i][0]];
          if (typeof fn === 'function') {
            results.push(parseFloat(fn.apply(Math, tests[i][1]).toPrecision(12)));
          }
        } catch (e) { results.push(0); }
      }
      return results;
    }

    function renderTable(data) {
      var tbody = document.getElementById('fp-tbody');
      var items = [
        ['User Agent', data.user_agent.substring(0, 80) + '...'],
        ['Platform', data.platform],
        ['Screen', data.screen_width + 'x' + data.screen_height + ' @ ' + data.screen_pixel_ratio + 'x'],
        ['Color Depth', data.screen_color_depth + ' bits'],
        ['Timezone', 'UTC' + (data.timezone_offset > 0 ? '-' : '+') + Math.abs(data.timezone_offset) + ' (' + data.timezone_name + ')'],
        ['Languages', data.languages.join(', ')],
        ['WebGL Vendor', data.webgl_vendor || 'N/A'],
        ['WebGL Renderer', data.webgl_renderer || 'N/A'],
        ['Canvas', data.canvas],
        ['Audio Hash', data.audio_hash || 'N/A'],
        ['Fonts', data.fonts.length + ' detected: ' + data.fonts.slice(0, 5).join(', ') + (data.fonts.length > 5 ? '...' : '')],
        ['Touch Points', data.touch_max_points],
        ['CPU Cores', data.hardware_concurrency],
        ['Device Memory', data.device_memory + ' GB'],
        ['Math Results', data.math_results.slice(0, 5).join(', ') + '...']
      ];
      tbody.innerHTML = items.map(function(row) {
        return '<tr><td><strong>' + row[0] + '</strong></td><td>' + row[1] + '</td></tr>';
      }).join('');
    }

    setTimeout(function() {
      var data = collectFingerprint();
      document.getElementById('fingerprint-data').textContent = JSON.stringify(data, null, 2);
      document.getElementById('status').className = 'status ready';
      document.getElementById('status').textContent = 'Fingerprint collected successfully';
      document.getElementById('result').style.display = 'block';
      document.getElementById('table').style.display = 'block';
      renderTable(data);
      fetch('/api/fingerprint', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      }).catch(function(err) { console.log('Failed to send to server:', err); });
    }, 500);
  </script>
</body>
</html>'''


class FingerprintHandler(BaseHTTPRequestHandler):
    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type')
        self.end_headers()

    def do_GET(self):
        parsed = urlparse(self.path)

        if parsed.path == '/':
            self.send_response(200)
            self.send_header('Content-Type', 'text/html; charset=utf-8')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            self.wfile.write(FINGERPRINT_HTML.encode())
            return

        if parsed.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            self.wfile.write(json.dumps({'status': 'ok', 'port': PORT}).encode())
            return

        if parsed.path == '/api/fingerprints':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            self.wfile.write(json.dumps(fingerprints).encode())
            return

        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        parsed = urlparse(self.path)

        if parsed.path == '/api/fingerprint':
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length).decode()

            try:
                data = json.loads(body)
                key = data.get('user_agent', 'unknown')
                fingerprints[key] = {
                    'data': data,
                    'created_at': time.time()
                }
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                self.wfile.write(json.dumps({'status': 'ok'}).encode())
            except json.JSONDecodeError:
                self.send_response(400)
                self.end_headers()
            return

        self.send_response(404)
        self.end_headers()

    def log_message(self, format, *args):
        print(f"[{self.log_date_time_string()}] {format % args}")


def main():
    server = HTTPServer(('0.0.0.0', PORT), FingerprintHandler)
    print(f"Fingerprint server starting on port {PORT}")
    print(f"Test page: http://localhost:{PORT}/")
    print(f"API endpoint: http://localhost:{PORT}/api/fingerprint")
    print(f"Press Ctrl+C to stop")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down...")
        server.shutdown()


if __name__ == '__main__':
    main()
