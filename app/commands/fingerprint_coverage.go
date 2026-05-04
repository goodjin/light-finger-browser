package commands

type FingerprintCoverageReport struct {
	ActiveEngine string                     `json:"active_engine"`
	Fields       []*FingerprintCoverageItem `json:"fields"`
}

type FingerprintCoverageItem struct {
	Field        string `json:"field"`
	LocalChrome  string `json:"local_chrome"`
	CloakBrowser string `json:"cloakbrowser"`
	Notes        string `json:"notes"`
}

func GetFingerprintCoverageReport() *FingerprintCoverageReport {
	cfg := loadBrowserEngineConfig()
	return &FingerprintCoverageReport{
		ActiveEngine: string(cfg.Engine),
		Fields: []*FingerprintCoverageItem{
			{
				Field:        "seed",
				LocalChrome:  "metadata-only",
				CloakBrowser: "metadata-only",
				Notes:        "Seed drives generation but is not injected into runtime directly.",
			},
			{
				Field:        "user_agent",
				LocalChrome:  "launch-arg",
				CloakBrowser: "launch-arg",
				Notes:        "Injected as --user-agent.",
			},
			{
				Field:        "platform",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "Local Chrome path does not currently override navigator.platform.",
			},
			{
				Field:        "screen.width",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "CloakBrowser receives screen width/height/pixel ratio at startup.",
			},
			{
				Field:        "screen.height",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "CloakBrowser receives screen width/height/pixel ratio at startup.",
			},
			{
				Field:        "screen.pixel_ratio",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "CloakBrowser receives screen width/height/pixel ratio at startup.",
			},
			{
				Field:        "timezone",
				LocalChrome:  "cdp-override",
				CloakBrowser: "launch-arg+cdp-override",
				Notes:        "Both paths reinforce timezone through CDP override after startup.",
			},
			{
				Field:        "locale",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "Local Chrome path does not currently inject locale/language settings.",
			},
			{
				Field:        "canvas.hash",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "Local path cannot currently influence canvas fingerprint output.",
			},
			{
				Field:        "webgl.vendor",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "Only CloakBrowser path wires vendor/renderer startup flags today.",
			},
			{
				Field:        "webgl.renderer",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "Only CloakBrowser path wires vendor/renderer startup flags today.",
			},
			{
				Field:        "webgl.extensions",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "Generated extensions are not injected by either runtime path today.",
			},
			{
				Field:        "audio.hash",
				LocalChrome:  "unsupported",
				CloakBrowser: "launch-arg",
				Notes:        "Audio fingerprint wiring exists only on the CloakBrowser startup path.",
			},
			{
				Field:        "hardware.cpu_cores",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "Hardware overrides are generated but not yet applied by runtime managers.",
			},
			{
				Field:        "hardware.memory_gb",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "Hardware overrides are generated but not yet applied by runtime managers.",
			},
			{
				Field:        "hardware.gpu_vendor",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "GPU metadata exists in the model but is not separately injected beyond WebGL flags.",
			},
			{
				Field:        "hardware.gpu_model",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "GPU model is not applied by either runtime path today.",
			},
			{
				Field:        "hardware.gpu_renderer",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "GPU renderer field is not injected separately from WebGL renderer.",
			},
			{
				Field:        "network.connection_type",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "Network shaping values are generated but not injected by runtime managers.",
			},
			{
				Field:        "network.downlink",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "Network shaping values are generated but not injected by runtime managers.",
			},
			{
				Field:        "network.rtt",
				LocalChrome:  "unsupported",
				CloakBrowser: "unsupported",
				Notes:        "Network shaping values are generated but not injected by runtime managers.",
			},
		},
	}
}
