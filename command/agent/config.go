package agent

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl"
	hclobj "github.com/hashicorp/hcl/hcl"
	client "github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad"
)

// Config is the configuration for the Nomad agent.
type Config struct {
	// Region is the region this agent is in. Defaults to global.
	Region string `hcl:"region"`

	// Datacenter is the datacenter this agent is in. Defaults to dc1
	Datacenter string `hcl:"datacenter"`

	// NodeName is the name we register as. Defaults to hostname.
	NodeName string `hcl:"name"`

	// DataDir is the directory to store our state in
	DataDir string `hcl:"data_dir"`

	// LogLevel is the level of the logs to putout
	LogLevel string `hcl:"log_level"`

	// BindAddr is the address on which all of nomad's services will
	// be bound. If not specified, this defaults to 127.0.0.1.
	BindAddr string `hcl:"bind_addr"`

	// EnableDebug is used to enable debugging HTTP endpoints
	EnableDebug bool `hcl:"enable_debug"`

	// Ports is used to control the network ports we bind to.
	Ports *Ports `hcl:"ports"`

	// Addresses is used to override the network addresses we bind to.
	Addresses *Addresses `hcl:"addresses"`

	// AdvertiseAddrs is used to control the addresses we advertise.
	AdvertiseAddrs *AdvertiseAddrs `hcl:"advertise"`

	// Client has our client related settings
	Client *ClientConfig `hcl:"client"`

	// Server has our server related settings
	Server *ServerConfig `hcl:"server"`

	// Telemetry is used to configure sending telemetry
	Telemetry *Telemetry `hcl:"telemetry"`

	// LeaveOnInt is used to gracefully leave on the interrupt signal
	LeaveOnInt bool `hcl:"leave_on_interrupt"`

	// LeaveOnTerm is used to gracefully leave on the terminate signal
	LeaveOnTerm bool `hcl:"leave_on_terminate"`

	// EnableSyslog is used to enable sending logs to syslog
	EnableSyslog bool `hcl:"enable_syslog"`

	// SyslogFacility is used to control the syslog facility used.
	SyslogFacility string `hcl:"syslog_facility"`

	// DisableUpdateCheck is used to disable the periodic update
	// and security bulletin checking.
	DisableUpdateCheck bool `hcl:"disable_update_check"`

	// DisableAnonymousSignature is used to disable setting the
	// anonymous signature when doing the update check and looking
	// for security bulletins
	DisableAnonymousSignature bool `hcl:"disable_anonymous_signature"`

	// AtlasConfig is used to configure Atlas
	Atlas *AtlasConfig `hcl:"atlas"`

	// NomadConfig is used to override the default config.
	// This is largly used for testing purposes.
	NomadConfig *nomad.Config `hcl:"-" json:"-"`

	// ClientConfig is used to override the default config.
	// This is largly used for testing purposes.
	ClientConfig *client.Config `hcl:"-" json:"-"`

	// DevMode is set by the -dev CLI flag.
	DevMode bool `hcl:"-"`

	// Version information is set at compilation time
	Revision          string
	Version           string
	VersionPrerelease string
}

// AtlasConfig is used to enable an parameterize the Atlas integration
type AtlasConfig struct {
	// Infrastructure is the name of the infrastructure
	// we belong to. e.g. hashicorp/stage
	Infrastructure string `hcl:"infrastructure"`

	// Token is our authentication token from Atlas
	Token string `hcl:"token" json:"-"`

	// Join controls if Atlas will attempt to auto-join the node
	// to it's cluster. Requires Atlas integration.
	Join bool `hcl:"join"`

	// Endpoint is the SCADA endpoint used for Atlas integration. If
	// empty, the defaults from the provider are used.
	Endpoint string `hcl:"endpoint"`
}

// ClientConfig is configuration specific to the client mode
type ClientConfig struct {
	// Enabled controls if we are a client
	Enabled bool `hcl:"enabled"`

	// StateDir is the state directory
	StateDir string `hcl:"state_dir"`

	// AllocDir is the directory for storing allocation data
	AllocDir string `hcl:"alloc_dir"`

	// Servers is a list of known server addresses. These are as "host:port"
	Servers []string `hcl:"servers"`

	// NodeID is the unique node identifier to use. A UUID is used
	// if not provided, and stored in the data directory
	NodeID string `hcl:"node_id"`

	// NodeClass is used to group the node by class
	NodeClass string `hcl:"node_class"`

	// Metadata associated with the node
	Meta map[string]string `hcl:"meta"`
}

// ServerConfig is configuration specific to the server mode
type ServerConfig struct {
	// Enabled controls if we are a server
	Enabled bool `hcl:"enabled"`

	// BootstrapExpect tries to automatically bootstrap the Consul cluster,
	// by witholding peers until enough servers join.
	BootstrapExpect int `hcl:"bootstrap_expect"`

	// DataDir is the directory to store our state in
	DataDir string `hcl:"data_dir"`

	// ProtocolVersion is the protocol version to speak. This must be between
	// ProtocolVersionMin and ProtocolVersionMax.
	ProtocolVersion int `hcl:"protocol_version"`

	// NumSchedulers is the number of scheduler thread that are run.
	// This can be as many as one per core, or zero to disable this server
	// from doing any scheduling work.
	NumSchedulers int `hcl:"num_schedulers"`

	// EnabledSchedulers controls the set of sub-schedulers that are
	// enabled for this server to handle. This will restrict the evaluations
	// that the workers dequeue for processing.
	EnabledSchedulers []string `hcl:"enabled_schedulers"`
}

// Telemetry is the telemetry configuration for the server
type Telemetry struct {
	StatsiteAddr    string `hcl:"statsite_address"`
	StatsdAddr      string `hcl:"statsd_address"`
	DisableHostname bool   `hcl:"disable_hostname"`
}

// Ports is used to encapsulate the various ports we bind to for network
// services. If any are not specified then the defaults are used instead.
type Ports struct {
	HTTP int `hcl:"http"`
	RPC  int `hcl:"rpc"`
	Serf int `hcl:"serf"`
}

// Addresses encapsulates all of the addresses we bind to for various
// network services. Everything is optional and defaults to BindAddr.
type Addresses struct {
	HTTP string `hcl:"http"`
	RPC  string `hcl:"rpc"`
	Serf string `hcl:"serf"`
}

// AdvertiseAddrs is used to control the addresses we advertise out for
// different network services. Not all network services support an
// advertise address. All are optional and default to BindAddr.
type AdvertiseAddrs struct {
	RPC  string `hcl:"rpc"`
	Serf string `hcl:"serf"`
}

// DevConfig is a Config that is used for dev mode of Nomad.
func DevConfig() *Config {
	conf := DefaultConfig()
	conf.LogLevel = "DEBUG"
	conf.Client.Enabled = true
	conf.Server.Enabled = true
	conf.DevMode = true
	conf.EnableDebug = true
	conf.DisableAnonymousSignature = true
	return conf
}

// DefaultConfig is a the baseline configuration for Nomad
func DefaultConfig() *Config {
	return &Config{
		LogLevel:   "INFO",
		Region:     "global",
		Datacenter: "dc1",
		BindAddr:   "127.0.0.1",
		Ports: &Ports{
			HTTP: 4646,
			RPC:  4647,
			Serf: 4648,
		},
		Addresses:      &Addresses{},
		AdvertiseAddrs: &AdvertiseAddrs{},
		Atlas:          &AtlasConfig{},
		Client: &ClientConfig{
			Enabled: false,
		},
		Server: &ServerConfig{
			Enabled: false,
		},
	}
}

// GetListener can be used to get a new listener using a custom bind address.
// If the bind provided address is empty, the BindAddr is used instead.
func (c *Config) Listener(proto, addr string, port int) (net.Listener, error) {
	if addr == "" {
		addr = c.BindAddr
	}
	return net.Listen(proto, fmt.Sprintf("%s:%d", addr, port))
}

// Merge merges two configurations.
func (a *Config) Merge(b *Config) *Config {
	var result Config = *a

	if b.Region != "" {
		result.Region = b.Region
	}
	if b.Datacenter != "" {
		result.Datacenter = b.Datacenter
	}
	if b.NodeName != "" {
		result.NodeName = b.NodeName
	}
	if b.DataDir != "" {
		result.DataDir = b.DataDir
	}
	if b.LogLevel != "" {
		result.LogLevel = b.LogLevel
	}
	if b.BindAddr != "" {
		result.BindAddr = b.BindAddr
	}
	if b.EnableDebug {
		result.EnableDebug = true
	}
	if b.LeaveOnInt {
		result.LeaveOnInt = true
	}
	if b.LeaveOnTerm {
		result.LeaveOnTerm = true
	}
	if b.EnableSyslog {
		result.EnableSyslog = true
	}
	if b.SyslogFacility != "" {
		result.SyslogFacility = b.SyslogFacility
	}
	if b.DisableUpdateCheck {
		result.DisableUpdateCheck = true
	}
	if b.DisableAnonymousSignature {
		result.DisableAnonymousSignature = true
	}

	// Apply the telemetry config
	if result.Telemetry == nil && b.Telemetry != nil {
		telemetry := *b.Telemetry
		result.Telemetry = &telemetry
	} else if b.Telemetry != nil {
		result.Telemetry = result.Telemetry.Merge(b.Telemetry)
	}

	// Apply the client config
	if result.Client == nil && b.Client != nil {
		client := *b.Client
		result.Client = &client
	} else if b.Client != nil {
		result.Client = result.Client.Merge(b.Client)
	}

	// Apply the server config
	if result.Server == nil && b.Server != nil {
		server := *b.Server
		result.Server = &server
	} else if b.Server != nil {
		result.Server = result.Server.Merge(b.Server)
	}

	// Apply the ports config
	if result.Ports == nil && b.Ports != nil {
		ports := *b.Ports
		result.Ports = &ports
	} else if b.Ports != nil {
		result.Ports = result.Ports.Merge(b.Ports)
	}

	// Apply the address config
	if result.Addresses == nil && b.Addresses != nil {
		addrs := *b.Addresses
		result.Addresses = &addrs
	} else if b.Addresses != nil {
		result.Addresses = result.Addresses.Merge(b.Addresses)
	}

	// Apply the advertise addrs config
	if result.AdvertiseAddrs == nil && b.AdvertiseAddrs != nil {
		advertise := *b.AdvertiseAddrs
		result.AdvertiseAddrs = &advertise
	} else if b.AdvertiseAddrs != nil {
		result.AdvertiseAddrs = result.AdvertiseAddrs.Merge(b.AdvertiseAddrs)
	}

	return &result
}

// Merge is used to merge two server configs together
func (a *ServerConfig) Merge(b *ServerConfig) *ServerConfig {
	var result ServerConfig = *a

	if b.Enabled {
		result.Enabled = true
	}
	if b.BootstrapExpect > 0 {
		result.BootstrapExpect = b.BootstrapExpect
	}
	if b.DataDir != "" {
		result.DataDir = b.DataDir
	}
	if b.ProtocolVersion != 0 {
		result.ProtocolVersion = b.ProtocolVersion
	}
	if b.NumSchedulers != 0 {
		result.NumSchedulers = b.NumSchedulers
	}

	// Add the schedulers
	result.EnabledSchedulers = append(result.EnabledSchedulers, b.EnabledSchedulers...)

	return &result
}

// Merge is used to merge two client configs together
func (a *ClientConfig) Merge(b *ClientConfig) *ClientConfig {
	var result ClientConfig = *a

	if b.Enabled {
		result.Enabled = true
	}
	if b.StateDir != "" {
		result.StateDir = b.StateDir
	}
	if b.AllocDir != "" {
		result.AllocDir = b.AllocDir
	}
	if b.NodeID != "" {
		result.NodeID = b.NodeID
	}
	if b.NodeClass != "" {
		result.NodeClass = b.NodeClass
	}

	// Add the servers
	result.Servers = append(result.Servers, b.Servers...)

	// Add the meta map values
	if result.Meta == nil {
		result.Meta = make(map[string]string)
	}
	for k, v := range b.Meta {
		result.Meta[k] = v
	}

	return &result
}

// Merge is used to merge two telemetry configs together
func (a *Telemetry) Merge(b *Telemetry) *Telemetry {
	var result Telemetry = *a

	if b.StatsiteAddr != "" {
		result.StatsiteAddr = b.StatsiteAddr
	}
	if b.StatsdAddr != "" {
		result.StatsdAddr = b.StatsdAddr
	}
	if b.DisableHostname {
		result.DisableHostname = true
	}
	return &result
}

// Merge is used to merge two port configurations.
func (a *Ports) Merge(b *Ports) *Ports {
	var result Ports = *a

	if b.HTTP != 0 {
		result.HTTP = b.HTTP
	}
	if b.RPC != 0 {
		result.RPC = b.RPC
	}
	if b.Serf != 0 {
		result.Serf = b.Serf
	}
	return &result
}

// Merge is used to merge two address configs together.
func (a *Addresses) Merge(b *Addresses) *Addresses {
	var result Addresses = *a

	if b.HTTP != "" {
		result.HTTP = b.HTTP
	}
	if b.RPC != "" {
		result.RPC = b.RPC
	}
	if b.Serf != "" {
		result.Serf = b.Serf
	}
	return &result
}

// Merge merges two advertise addrs configs together.
func (a *AdvertiseAddrs) Merge(b *AdvertiseAddrs) *AdvertiseAddrs {
	var result AdvertiseAddrs = *a

	if b.RPC != "" {
		result.RPC = b.RPC
	}
	if b.Serf != "" {
		result.Serf = b.Serf
	}
	return &result
}

// LoadConfig loads the configuration at the given path, regardless if
// its a file or directory.
func LoadConfig(path string) (*Config, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return LoadConfigDir(path)
	} else {
		return LoadConfigFile(path)
	}
}

// LoadConfigString is used to parse a config string
func LoadConfigString(s string) (*Config, error) {
	// Parse!
	obj, err := hcl.Parse(s)
	if err != nil {
		return nil, err
	}

	// Start building the result
	var result Config
	if err := hcl.DecodeObject(&result, obj); err != nil {
		return nil, err
	}

	return &result, nil
}

// LoadConfigFile loads the configuration from the given file.
func LoadConfigFile(path string) (*Config, error) {
	// Read the file
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadConfigString(string(d))
}

func getString(o *hclobj.Object) string {
	if o == nil || o.Type != hclobj.ValueTypeString {
		return ""
	}

	return o.Value.(string)
}

// LoadConfigDir loads all the configurations in the given directory
// in alphabetical order.
func LoadConfigDir(dir string) (*Config, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf(
			"configuration path must be a directory: %s",
			dir)
	}

	var files []string
	err = nil
	for err != io.EOF {
		var fis []os.FileInfo
		fis, err = f.Readdir(128)
		if err != nil && err != io.EOF {
			return nil, err
		}

		for _, fi := range fis {
			// Ignore directories
			if fi.IsDir() {
				continue
			}

			// Only care about files that are valid to load.
			name := fi.Name()
			skip := true
			if strings.HasSuffix(name, ".hcl") {
				skip = false
			} else if strings.HasSuffix(name, ".json") {
				skip = false
			}
			if skip || isTemporaryFile(name) {
				continue
			}

			path := filepath.Join(dir, name)
			files = append(files, path)
		}
	}

	// Fast-path if we have no files
	if len(files) == 0 {
		return &Config{}, nil
	}

	var result *Config
	for _, f := range files {
		config, err := LoadConfigFile(f)
		if err != nil {
			return nil, fmt.Errorf("Error loading %s: %s", f, err)
		}

		if result == nil {
			result = config
		} else {
			result = result.Merge(config)
		}
	}

	return result, nil
}

// isTemporaryFile returns true or false depending on whether the
// provided file name is a temporary file for the following editors:
// emacs or vim.
func isTemporaryFile(name string) bool {
	return strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, ".#") || // emacs
		(strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#")) // emacs
}
