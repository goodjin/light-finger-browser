# Fingerprint Verification Results

## Overview

The fingerprint verification system compares browser-collected fingerprint data against expected values to detect inconsistencies that might indicate tracking or fingerprinting.

## Verification Components

| Component | Weight | Description |
|-----------|--------|-------------|
| Canvas | 20 | Text and geometry rendering fingerprint |
| WebGL | 18 | GPU vendor and renderer information |
| Audio | 15 | AudioContext hash based on audio processing |
| Fonts | 12 | Detected system fonts via dimension-based detection |
| Screen | 8 | Resolution, color depth, pixel ratio |
| Timezone | 5 | Offset and Intl timezone name |
| Languages | 5 | navigator.languages array |
| Math | 8 | 22-function result vector for JS engine differentiation |
| Touch Support | 3 | maxTouchPoints capability |
| Platform | 3 | navigator.platform value |
| Hardware Concurrency | 2 | CPU core count |
| Device Memory | 3 | navigator.deviceMemory value |

## Match Score Calculation

- **Exact string match**: 100%
- **Numeric with tolerance**: percentage based on relative difference
- **Array (fonts/languages)**: Jaccard similarity
- **Object (WebGL params)**: average of vendor and renderer similarity

## Quality Ratings

| Score Range | Rating |
|-------------|--------|
| 90-100% | Excellent |
| 70-89% | Good |
| 50-69% | Poor |
| 0-49% | Fail |

## Usage

1. Start the mock fingerprint server on port 18080
2. Launch a browser that loads the fingerprint test page
3. The test page collects browser fingerprint components via JavaScript
4. Fingerprint data is sent to the mock server via POST `/api/fingerprint`
5. View detailed field-by-field comparison results via the UI

## Mock Server Endpoints

- `GET /` - Serves fingerprint test HTML page
- `POST /api/fingerprint` - Receives fingerprint data
- `GET /api/fingerprints` - Returns stored fingerprints
- `GET /health` - Health check endpoint

## Test Page Fingerprint Collection

The `fingerprint-test.html` page collects:

1. **Canvas**: Renders text and geometry, returns data URL
2. **WebGL**: Queries `WEBGL_debug_renderer_info` extension
3. **Audio**: Creates OfflineAudioContext hash
4. **Fonts**: Uses dimension-based detection with base fonts
5. **Screen**: Reads `screen` object properties
6. **Timezone**: Gets `getTimezoneOffset()` and `Intl.DateTimeFormat`
7. **Languages**: Reads `navigator.languages`
8. **Math**: Tests 22 Math functions for consistent results
9. **Touch**: Reads `navigator.maxTouchPoints`
10. **Platform**: Reads `navigator.platform`
11. **HardwareConcurrency**: Reads `navigator.hardwareConcurrency`
12. **DeviceMemory**: Reads `navigator.deviceMemory`

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   React UI      │────>│  Go Backend      │────>│  Mock Server    │
│  (Frontend)     │     │  (Wails)         │     │  (Port 18080)  │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                                          │
                                                          v
                                                  ┌─────────────────┐
                                                  │   Browser       │
                                                  │  (Fingerprint   │
                                                  │   Collection)   │
                                                  └─────────────────┘
```

## Files

- `cmd/fingerprint-server/main.go` - Mock server implementation
- `frontend/src/fingerprint-test.html` - Plain JS fingerprint extraction
- `frontend/src/components/FingerprintTestPage.tsx` - React display component
- `app/commands/fingerprint_server.go` - Go service layer
