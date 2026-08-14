package config

import (
	"encoding/json"
	"os"
)

// Config holds runtime configuration for detection and app behavior.
// Fields may be loaded from a JSON file and overridden by command-line flags.
type Config struct {
	Debug bool `json:"debug"`
	// Detection parameters
	MinScale       float64 `json:"min_scale"`
	MaxScale       float64 `json:"max_scale"`
	ScaleStep      float64 `json:"scale_step"`
	Threshold      float64 `json:"threshold"`
	Stride         int     `json:"stride"`
	Refine         bool    `json:"refine"`
	StopOnScore    float64 `json:"stop_on_score"`
	ReturnBestEven bool    `json:"return_best_even"`

	// Selection rectangle persistence (Phase2)
	SelectionX int `json:"selection_x"`
	SelectionY int `json:"selection_y"`
	SelectionW int `json:"selection_w"`
	SelectionH int `json:"selection_h"`

	// Reel key configuration (e.g. "F3" or "R")
	ReelKey string `json:"reel_key"`

	// Bite detection configuration (only actively used fields retained).
	ROISizePx int `json:"roi_size_px"` // square ROI side length in pixels
	// MaxCastDurationSeconds defines the maximum expected lifetime of a fishing cast (bobber present).
	// If monitoring exceeds this duration, the target is considered lost and the system returns to searching.
	MaxCastDurationSeconds int `json:"max_cast_duration_seconds"`
	// CooldownSeconds defines how long to wait after reeling before attempting the next cast.
	CooldownSeconds int `json:"cooldown_seconds"`

	// AnalysisScale optionally downsizes frames before expensive template matching.
	// Range (0.2 - 1.0]. 1.0 means disabled. Smaller values reduce CPU at the cost of precision.
	AnalysisScale float64 `json:"analysis_scale"`

	// DarkMode persists user preference for dark theme across sessions.
	DarkMode bool `json:"dark_mode"`

	// UseColorDetect switches detection from NCC template matching to
	// hue-based blob detection. When true, MinScale/MaxScale/ScaleStep/
	// Threshold/Stride/Refine/StopOnScore/ReturnBestEven are ignored.
	UseColorDetect bool `json:"use_color_detect"`
	// ColorRedDelta is how much R must exceed B for a red feather pixel.
	ColorRedDelta int `json:"color_red_delta"`
	// ColorBlueDelta is how much B must exceed R for a blue feather pixel.
	ColorBlueDelta int `json:"color_blue_delta"`
	// ColorMinValue is the minimum channel brightness; rejects dark noise.
	ColorMinValue int `json:"color_min_value"`
	// ColorMinPixels is the smallest blob accepted as a bobber.
	ColorMinPixels int `json:"color_min_pixels"`
	// ColorMaxPixels is the largest blob accepted; rejects large regions.
	ColorMaxPixels int `json:"color_max_pixels"`
}

// Accessor helpers to satisfy fishing.ConfigLite without exposing struct embedding.

// DefaultConfig returns a Config populated with standard defaults.
func DefaultConfig() *Config {
	return &Config{
		Debug:                  false,
		MinScale:               0.90, // from pixle_bot_config.json
		MaxScale:               1.20, // from pixle_bot_config.json
		ScaleStep:              0.05, // from pixle_bot_config.json
		Threshold:              0.73, // from pixle_bot_config.json
		Stride:                 6,    // from pixle_bot_config.json
		Refine:                 true,
		StopOnScore:            0.80, // from pixle_bot_config.json
		ReturnBestEven:         true,
		SelectionX:             0, // persisted selection defaults
		SelectionY:             0,
		SelectionW:             0,
		SelectionH:             0,
		ReelKey:                "F3",
		ROISizePx:              80,
		MaxCastDurationSeconds: 16,
		CooldownSeconds:        8, // from pixle_bot_config.json
		AnalysisScale:          1.0,
		DarkMode:               true, // from pixle_bot_config.json
		UseColorDetect:         true,
		ColorRedDelta:          60,
		ColorBlueDelta:         50,
		ColorMinValue:          100,
		ColorMinPixels:         25,
		ColorMaxPixels:         4000,
	}
}

// Validate clamps/normalizes values to safe ranges.
func (c *Config) Validate() error {
	if c.MinScale <= 0 {
		c.MinScale = 0.60
	}
	if c.MaxScale <= 0 || c.MaxScale < c.MinScale {
		c.MaxScale = c.MinScale + 0.80
	}
	if c.ScaleStep <= 0 {
		c.ScaleStep = 0.05
	}
	if c.ScaleStep > (c.MaxScale - c.MinScale) {
		c.ScaleStep = (c.MaxScale - c.MinScale) / 4
	}
	if c.Threshold <= 0 || c.Threshold > 1 {
		c.Threshold = 0.80
	}
	if c.Stride <= 0 {
		c.Stride = 4
	}
	if c.StopOnScore < 0 || c.StopOnScore > 1 {
		c.StopOnScore = 0.95
	}
	if c.ReelKey == "" {
		c.ReelKey = "F3"
	}
	// Bite detection validation & sane clamps
	if c.ROISizePx < 32 {
		c.ROISizePx = 32
	}
	if c.ROISizePx > 256 { // keep ROI modest for performance
		c.ROISizePx = 256
	}

	if c.MaxCastDurationSeconds < 5 { // extremely short casts are unlikely; enforce reasonable floor
		c.MaxCastDurationSeconds = 5
	}
	if c.MaxCastDurationSeconds > 180 { // safety upper bound (3 minutes) though typical is ~30s
		c.MaxCastDurationSeconds = 180
	}

	// Cooldown seconds sanity (allow zero -> default minimal, clamp upper bound for safety)
	if c.CooldownSeconds <= 0 {
		c.CooldownSeconds = 1
	}
	if c.CooldownSeconds > 60 { // more than a minute likely unnecessary
		c.CooldownSeconds = 60
	}

	// AnalysisScale validation
	if c.AnalysisScale <= 0 {
		c.AnalysisScale = 1.0
	}
	if c.AnalysisScale < 0.2 {
		c.AnalysisScale = 0.2
	}
	if c.AnalysisScale > 1.0 {
		c.AnalysisScale = 1.0
	}

	// Color detection clamps.
	if c.ColorRedDelta <= 0 {
		c.ColorRedDelta = 60
	}
	if c.ColorBlueDelta <= 0 {
		c.ColorBlueDelta = 50
	}
	if c.ColorMinValue <= 0 {
		c.ColorMinValue = 100
	}
	if c.ColorMinPixels <= 0 {
		c.ColorMinPixels = 25
	}
	if c.ColorMaxPixels <= c.ColorMinPixels {
		c.ColorMaxPixels = c.ColorMinPixels * 160
	}

	return nil
}

// Load attempts to read configuration from the given JSON file path. If the file does not
// exist it returns DefaultConfig(). On JSON error it returns defaults with the error.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if err := dec.Decode(cfg); err != nil {
		return cfg, err
	}
	_ = cfg.Validate()
	return cfg, nil
}

// Save writes the configuration to the given path in JSON format.
func (c *Config) Save(path string) error {
	_ = c.Validate()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}
