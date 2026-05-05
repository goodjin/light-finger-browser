package commands

type FingerprintCoverageReport struct {
	ActiveEngine string                     `json:"active_engine"`
	Fields       []*FingerprintCoverageItem `json:"fields"`
}

type FingerprintCoverageItem struct {
	Field       string `json:"field"`
	LocalChrome string `json:"local_chrome"`
	SelfBuilt   string `json:"self_built"`
	Notes       string `json:"notes"`
}

func GetFingerprintCoverageReport() *FingerprintCoverageReport {
	cfg := loadBrowserEngineConfig()
	return &FingerprintCoverageReport{
		ActiveEngine: string(cfg.Engine),
		Fields: []*FingerprintCoverageItem{
			{
				Field:       "seed",
				LocalChrome: "metadata-only",
				SelfBuilt:   "metadata-only",
				Notes:       "Seed drives generation but is not injected into runtime directly.",
			},
			{
				Field:       "user_agent",
				LocalChrome: "launch-arg",
				SelfBuilt:   "launch-arg",
				Notes:       "Injected as --user-agent.",
			},
			{
				Field:       "platform",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Local Chrome path does not currently override navigator.platform.",
			},
			{
				Field:       "screen.width",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Self-built browser receives screen width/height/pixel ratio at startup.",
			},
			{
				Field:       "screen.height",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Self-built browser receives screen width/height/pixel ratio at startup.",
			},
			{
				Field:       "screen.pixel_ratio",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Self-built browser receives screen width/height/pixel ratio at startup.",
			},
			{
				Field:       "timezone",
				LocalChrome: "cdp-override",
				SelfBuilt:   "launch-arg+cdp-override",
				Notes:       "Both paths reinforce timezone through CDP override after startup.",
			},
			{
				Field:       "locale",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Local Chrome path does not currently inject locale/language settings.",
			},
			{
				Field:       "canvas.hash",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Local path cannot currently influence canvas fingerprint output.",
			},
			{
				Field:       "webgl.vendor",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Only self-built browser path wires vendor/renderer startup flags today.",
			},
			{
				Field:       "webgl.renderer",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Only self-built browser path wires vendor/renderer startup flags today.",
			},
			{
				Field:       "webgl.extensions",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "Generated extensions are not injected by either runtime path today.",
			},
			{
				Field:       "audio.hash",
				LocalChrome: "unsupported",
				SelfBuilt:   "launch-arg",
				Notes:       "Audio fingerprint wiring exists only on the self-built browser startup path.",
			},
			{
				Field:       "hardware.cpu_cores",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "Hardware overrides are generated but not yet applied by runtime managers.",
			},
			{
				Field:       "hardware.memory_gb",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "Hardware overrides are generated but not yet applied by runtime managers.",
			},
			{
				Field:       "hardware.gpu_vendor",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "GPU metadata exists in the model but is not separately injected beyond WebGL flags.",
			},
			{
				Field:       "hardware.gpu_model",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "GPU model is not applied by either runtime path today.",
			},
			{
				Field:       "hardware.gpu_renderer",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "GPU renderer field is not injected separately from WebGL renderer.",
			},
			{
				Field:       "network.connection_type",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "Network shaping values are generated but not injected by runtime managers.",
			},
			{
				Field:       "network.downlink",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "Network shaping values are generated but not injected by runtime managers.",
			},
			{
				Field:       "network.rtt",
				LocalChrome: "unsupported",
				SelfBuilt:   "unsupported",
				Notes:       "Network shaping values are generated but not injected by runtime managers.",
			},
		},
	}
}
