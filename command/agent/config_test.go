package agent

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/nomad/nomad/structs"
)

func TestConfig_Merge(t *testing.T) {
	c1 := &Config{
		Region:                    "global",
		Datacenter:                "dc1",
		NodeName:                  "node1",
		DataDir:                   "/tmp/dir1",
		LogLevel:                  "INFO",
		EnableDebug:               false,
		LeaveOnInt:                false,
		LeaveOnTerm:               false,
		EnableSyslog:              false,
		SyslogFacility:            "local0.info",
		DisableUpdateCheck:        false,
		DisableAnonymousSignature: false,
		BindAddr:                  "127.0.0.1",
		Telemetry: &Telemetry{
			StatsiteAddr:    "127.0.0.1:8125",
			StatsdAddr:      "127.0.0.1:8125",
			DisableHostname: false,
		},
		Client: &ClientConfig{
			Enabled:   false,
			StateDir:  "/tmp/state1",
			AllocDir:  "/tmp/alloc1",
			NodeID:    "node1",
			NodeClass: "class1",
		},
		Server: &ServerConfig{
			Enabled:         false,
			BootstrapExpect: 1,
			DataDir:         "/tmp/data1",
			ProtocolVersion: 1,
			NumSchedulers:   1,
		},
		Ports: &Ports{
			HTTP: 4646,
			RPC:  4647,
			Serf: 4648,
		},
		Addresses: &Addresses{
			HTTP: "127.0.0.1",
			RPC:  "127.0.0.1",
			Serf: "127.0.0.1",
		},
		AdvertiseAddrs: &AdvertiseAddrs{
			RPC:  "127.0.0.1",
			Serf: "127.0.0.1",
		},
	}

	c2 := &Config{
		Region:                    "region2",
		Datacenter:                "dc2",
		NodeName:                  "node2",
		DataDir:                   "/tmp/dir2",
		LogLevel:                  "DEBUG",
		EnableDebug:               true,
		LeaveOnInt:                true,
		LeaveOnTerm:               true,
		EnableSyslog:              true,
		SyslogFacility:            "local0.debug",
		DisableUpdateCheck:        true,
		DisableAnonymousSignature: true,
		BindAddr:                  "127.0.0.2",
		Telemetry: &Telemetry{
			StatsiteAddr:    "127.0.0.2:8125",
			StatsdAddr:      "127.0.0.2:8125",
			DisableHostname: true,
		},
		Client: &ClientConfig{
			Enabled:   true,
			StateDir:  "/tmp/state2",
			AllocDir:  "/tmp/alloc2",
			NodeID:    "node2",
			NodeClass: "class2",
			Servers:   []string{"server2"},
			Meta:      map[string]string{"baz": "zip"},
		},
		Server: &ServerConfig{
			Enabled:           true,
			BootstrapExpect:   2,
			DataDir:           "/tmp/data2",
			ProtocolVersion:   2,
			NumSchedulers:     2,
			EnabledSchedulers: []string{structs.JobTypeBatch},
		},
		Ports: &Ports{
			HTTP: 20000,
			RPC:  21000,
			Serf: 22000,
		},
		Addresses: &Addresses{
			HTTP: "127.0.0.2",
			RPC:  "127.0.0.2",
			Serf: "127.0.0.2",
		},
		AdvertiseAddrs: &AdvertiseAddrs{
			RPC:  "127.0.0.2",
			Serf: "127.0.0.2",
		},
	}

	result := c1.Merge(c2)
	if !reflect.DeepEqual(result, c2) {
		t.Fatalf("bad:\n%#v\n%#v", result.Server, c2.Server)
	}
}

func TestConfig_LoadConfigFile(t *testing.T) {
	// Fails if the file doesn't exist
	if _, err := LoadConfigFile("/unicorns/leprechauns"); err == nil {
		t.Fatalf("expected error, got nothing")
	}

	fh, err := ioutil.TempFile("", "nomad")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(fh.Name())

	// Invalid content returns error
	if _, err := fh.WriteString("nope"); err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, err := LoadConfigFile(fh.Name()); err == nil {
		t.Fatalf("expected load error, got nothing")
	}

	// Valid content parses successfully
	if err := fh.Truncate(0); err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, err := fh.Seek(0, 0); err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, err := fh.WriteString(`{"region":"west"}`); err != nil {
		t.Fatalf("err: %s", err)
	}

	config, err := LoadConfigFile(fh.Name())
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config.Region != "west" {
		t.Fatalf("bad region: %q", config.Region)
	}
}

func TestConfig_LoadConfigDir(t *testing.T) {
	// Fails if the dir doesn't exist.
	if _, err := LoadConfigDir("/unicorns/leprechauns"); err == nil {
		t.Fatalf("expected error, got nothing")
	}

	dir, err := ioutil.TempDir("", "nomad")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)

	// Returns empty config on empty dir
	config, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config == nil {
		t.Fatalf("should not be nil")
	}

	file1 := filepath.Join(dir, "conf1.hcl")
	err = ioutil.WriteFile(file1, []byte(`{"region":"west"}`), 0600)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	file2 := filepath.Join(dir, "conf2.hcl")
	err = ioutil.WriteFile(file2, []byte(`{"datacenter":"sfo"}`), 0600)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	file3 := filepath.Join(dir, "conf3.hcl")
	err = ioutil.WriteFile(file3, []byte(`nope`), 0600)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Fails if we have a bad config file
	if _, err := LoadConfigDir(dir); err == nil {
		t.Fatalf("expected load error, got nothing")
	}

	if err := os.Remove(file3); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Works if configs are valid
	config, err = LoadConfigDir(dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config.Region != "west" || config.Datacenter != "sfo" {
		t.Fatalf("bad: %#v", config)
	}
}

func TestConfig_LoadConfig(t *testing.T) {
	// Fails if the target doesn't exist
	if _, err := LoadConfig("/unicorns/leprechauns"); err == nil {
		t.Fatalf("expected error, got nothing")
	}

	fh, err := ioutil.TempFile("", "nomad")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Remove(fh.Name())

	if _, err := fh.WriteString(`{"region":"west"}`); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Works on a config file
	config, err := LoadConfig(fh.Name())
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config.Region != "west" {
		t.Fatalf("bad: %#v", config)
	}

	dir, err := ioutil.TempDir("", "nomad")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)

	file1 := filepath.Join(dir, "config1.hcl")
	err = ioutil.WriteFile(file1, []byte(`{"datacenter":"sfo"}`), 0600)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Works on config dir
	config, err = LoadConfig(dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if config.Datacenter != "sfo" {
		t.Fatalf("bad: %#v", config)
	}
}

func TestConfig_Listener(t *testing.T) {
	config := DefaultConfig()

	// Fails on invalid input
	if _, err := config.Listener("tcp", "nope", 8080); err == nil {
		t.Fatalf("expected addr error")
	}
	if _, err := config.Listener("nope", "127.0.0.1", 8080); err == nil {
		t.Fatalf("expected protocol err")
	}
	if _, err := config.Listener("tcp", "127.0.0.1", -1); err == nil {
		t.Fatalf("expected port error")
	}

	// Works with valid inputs
	ln, err := config.Listener("tcp", "127.0.0.1", 24000)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	ln.Close()

	if net := ln.Addr().Network(); net != "tcp" {
		t.Fatalf("expected tcp, got: %q", net)
	}
	if addr := ln.Addr().String(); addr != "127.0.0.1:24000" {
		t.Fatalf("expected 127.0.0.1:4646, got: %q", addr)
	}

	// Falls back to default bind address if non provided
	config.BindAddr = "0.0.0.0"
	ln, err = config.Listener("tcp4", "", 24000)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	ln.Close()

	if addr := ln.Addr().String(); addr != "0.0.0.0:24000" {
		t.Fatalf("expected 0.0.0.0:24000, got: %q", addr)
	}
}

func TestConfig_LoadConfigString(t *testing.T) {
	// Load the config
	config, err := LoadConfigString(testConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Expected output
	expect := &Config{
		Region:      "foobar",
		Datacenter:  "dc2",
		NodeName:    "my-web",
		DataDir:     "/tmp/nomad",
		LogLevel:    "ERR",
		BindAddr:    "192.168.0.1",
		EnableDebug: true,
		Ports: &Ports{
			HTTP: 1234,
			RPC:  2345,
			Serf: 3456,
		},
		Addresses: &Addresses{
			HTTP: "127.0.0.1",
			RPC:  "127.0.0.2",
			Serf: "127.0.0.3",
		},
		AdvertiseAddrs: &AdvertiseAddrs{
			RPC:  "127.0.0.3",
			Serf: "127.0.0.4",
		},
		Client: &ClientConfig{
			Enabled:   true,
			StateDir:  "/tmp/client-state",
			AllocDir:  "/tmp/alloc",
			Servers:   []string{"a.b.c:80", "127.0.0.1:1234"},
			NodeID:    "xyz123",
			NodeClass: "linux-medium-64bit",
			Meta: map[string]string{
				"foo": "bar",
				"baz": "zip",
			},
		},
		Server: &ServerConfig{
			Enabled:           true,
			BootstrapExpect:   5,
			DataDir:           "/tmp/data",
			ProtocolVersion:   3,
			NumSchedulers:     2,
			EnabledSchedulers: []string{"test"},
		},
		Telemetry: &Telemetry{
			StatsiteAddr:    "127.0.0.1:1234",
			StatsdAddr:      "127.0.0.1:2345",
			DisableHostname: true,
		},
		LeaveOnInt:                true,
		LeaveOnTerm:               true,
		EnableSyslog:              true,
		SyslogFacility:            "LOCAL1",
		DisableUpdateCheck:        true,
		DisableAnonymousSignature: true,
		Atlas: &AtlasConfig{
			Infrastructure: "armon/test",
			Token:          "abcd",
			Join:           true,
			Endpoint:       "127.0.0.1:1234",
		},
	}

	// Check parsing
	if !reflect.DeepEqual(config, expect) {
		t.Fatalf("bad: got: %#v\nexpect: %#v", config, expect)
	}
}

const testConfig = `
region = "foobar"
datacenter = "dc2"
name = "my-web"
data_dir = "/tmp/nomad"
log_level = "ERR"
bind_addr = "192.168.0.1"
enable_debug = true
ports {
	http = 1234
	rpc = 2345
	serf = 3456
}
addresses {
	http = "127.0.0.1"
	rpc = "127.0.0.2"
	serf = "127.0.0.3"
}
advertise {
	rpc = "127.0.0.3"
	serf = "127.0.0.4"
}
client {
	enabled = true
	state_dir = "/tmp/client-state"
	alloc_dir = "/tmp/alloc"
	servers = ["a.b.c:80", "127.0.0.1:1234"]
	node_id = "xyz123"
	node_class = "linux-medium-64bit"
	meta {
		foo = "bar"
		baz = "zip"
	}
}
server {
	enabled = true
	bootstrap_expect = 5
	data_dir = "/tmp/data"
	protocol_version = 3
	num_schedulers = 2
	enabled_schedulers = ["test"]
}
telemetry {
	statsite_address = "127.0.0.1:1234"
	statsd_address = "127.0.0.1:2345"
	disable_hostname = true
}
leave_on_interrupt = true
leave_on_terminate = true
enable_syslog = true
syslog_facility = "LOCAL1"
disable_update_check = true
disable_anonymous_signature = true
atlas {
	infrastructure = "armon/test"
	token = "abcd"
	join = true
	endpoint = "127.0.0.1:1234"
}
`
