package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	// Cast key configuration (e.g. "R" or "F3"). Despite the name this key is
	// pressed on cast; reeling is a right-click at the detected coordinates.
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
	// ColorMinBluePixels is the smallest accepted blue cluster. Blue feathers
	// are scarcer than red, so this is deliberately lower.
	ColorMinBluePixels int `json:"color_min_blue_pixels"`
	// ColorMaxPairDistance is the maximum centroid separation in pixels
	// between the red and blue clusters for them to count as one bobber.
	ColorMaxPairDistance int `json:"color_max_pair_distance"`
	// ColorMaxPixels is the largest blob accepted; rejects large regions.
	ColorMaxPixels int `json:"color_max_pixels"`

	// CastSettleMs blocks detection for this many milliseconds after a cast,
	// so the fading previous bobber is not mistaken for the new one.
	CastSettleMs int `json:"cast_settle_ms"`
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
		ReelKey:                "R",
		ROISizePx:              120, // tuned: bobber must occupy a small fraction of the ROI
		MaxCastDurationSeconds: 25,  // tuned: WoW casts run ~20-30s
		CooldownSeconds:        8, // from pixle_bot_config.json
		AnalysisScale:          1.0,
		DarkMode:               true, // from pixle_bot_config.json
		UseColorDetect:         true,
		ColorRedDelta:          40,
		ColorBlueDelta:         35,
		ColorMinValue:          80,
		ColorMinPixels:         12,
		ColorMinBluePixels:     5,
		ColorMaxPairDistance:   40,
		ColorMaxPixels:         4000,
		CastSettleMs:           2000,
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
		c.ReelKey = "R"
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
		c.ColorRedDelta = 40
	}
	if c.ColorBlueDelta <= 0 {
		c.ColorBlueDelta = 35
	}
	if c.ColorMinValue <= 0 {
		c.ColorMinValue = 80
	}
	if c.ColorMinPixels <= 0 {
		c.ColorMinPixels = 12
	}
	if c.ColorMinBluePixels <= 0 {
		c.ColorMinBluePixels = 5
	}
	if c.ColorMaxPairDistance <= 0 {
		c.ColorMaxPairDistance = 40
	}
	if c.ColorMaxPixels <= c.ColorMinPixels {
		c.ColorMaxPixels = c.ColorMinPixels * 160
	}
	if c.CastSettleMs <= 0 {
		c.CastSettleMs = 2000
	}
	if c.CastSettleMs > 5000 {
		c.CastSettleMs = 5000
	}

	return nil
}

// ResolvePath returns the absolute location used to store the config file.
//
// A bare filename resolves to %APPDATA%\pixel-bot-go\<name> (or the platform
// equivalent) so settings survive downloading a new binary into a new folder.
// A file already present next to the executable takes priority, which keeps
// existing installs working. Paths containing a separator are used verbatim.
func ResolvePath(path string) string {
	if path == "" {
		path = "pixle_bot_config.json"
	}
	if filepath.Base(path) != path {
		return path
	}
	// Prefer an existing file in the working directory (legacy installs).
	if _, err := os.Stat(path); err == nil {
		return path
	}
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return path
	}
	appDir := filepath.Join(dir, "pixel-bot-go")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return path
	}
	return filepath.Join(appDir, path)
}

// Load attempts to read configuration from the given JSON file path. If the file does not
// exist it returns DefaultConfig(). On JSON error it returns defaults with the error.
func Load(path string) (*Config, error) {
	path = ResolvePath(path)
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
	path = ResolvePath(path)
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
