// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package agent

import (
	"os"
	"path"
	"path/filepath"

	"github.com/juju/errors"
	"github.com/juju/names/v4"
)

const (
	// AgentConfigFilename is the default file name of used for the agent
	// config.
	AgentConfigFilename = "agent.conf"
)

// The Config interface is the sole way that the agent gets access to the
// configuration information for the machine and unit agents.  There should
// only be one instance of a config object for any given agent, and this
// interface is passed between multiple go routines.  The mutable methods are
// protected by a mutex, and it is expected that the caller doesn't modify any
// slice that may be returned.
//
// NOTE: should new mutating methods be added to this interface, consideration
// is needed around the synchronisation as a single instance is used in
// multiple go routines.
type Config interface {
	// DataDir returns the data directory. Each agent has a subdirectory
	// containing the configuration files.
	DataDir() string

	// LogDir returns the log directory. All logs from all agents on
	// the machine are written to this directory.
	LogDir() string

	// CACert returns the CA certificate that is used to validate the state or
	// API server's certificate.
	CACert() string

	// APIAddresses returns the addresses needed to connect to the api server
	APIAddresses() ([]string, error)

	// StateServingInfo returns the details needed to run
	// a controller and reports whether those details
	// are available
	StateServingInfo() (StateServingInfo, bool)
}

// StateServingInfo holds network/auth information needed by a controller.
type StateServingInfo struct {
	APIPort           int
	ControllerAPIPort int
	Cert              string
	PrivateKey        string
	CAPrivateKey      string
	// this will be passed as the KeyFile argument to MongoDB
	SharedSecret   string
	SystemIdentity string
}

// BaseDir returns the directory containing the data directories for
// all the agents.
func BaseDir(dataDir string) string {
	// Note: must use path, not filepath, as this function is
	// (indirectly) used by the client on Windows.
	return path.Join(dataDir, "agents")
}

// Dir returns the agent-specific data directory.
func Dir(dataDir string, tag names.Tag) string {
	// Note: must use path, not filepath, as this
	// function is used by the client on Windows.
	return path.Join(BaseDir(dataDir), tag.String())
}

// ConfigPath returns the full path to the agent config file.
// NOTE: Delete this once all agents accept --config instead
// of --data-dir - it won't be needed anymore.
func ConfigPath(dataDir string, tag names.Tag) string {
	return filepath.Join(Dir(dataDir, tag), AgentConfigFilename)
}

// Paths holds the directory paths used by the agent.
type Paths struct {
	// DataDir is the data directory where each agent has a subdirectory
	// containing the configuration files.
	DataDir string
	// LogDir is the log directory where all logs from all agents on
	// the machine are written.
	LogDir string
	// ConfDir is the directory where all  config file for
	// Juju agents are stored.
	ConfDir string
}

var (
	// DefaultPaths defines the default paths for an agent.
	DefaultPaths = Paths{
		DataDir: DataDir(CurrentOS()),
		LogDir:  path.Join(LogDir(CurrentOS()), "juju"),
		ConfDir: ConfDir(CurrentOS()),
	}
)

// NewPathsWithDefaults returns a Paths struct initialized with default locations if not otherwise specified.
func NewPathsWithDefaults(p Paths) Paths {
	paths := DefaultPaths
	if p.DataDir != "" {
		paths.DataDir = p.DataDir
	}
	if p.LogDir != "" {
		paths.LogDir = p.LogDir
	}
	if p.ConfDir != "" {
		paths.ConfDir = p.ConfDir
	}
	return paths
}

type apiDetails struct {
	addresses []string
}

type configInternal struct {
	configFilePath string
	paths          Paths
	tag            names.Tag
	controller     names.ControllerTag
	model          names.ModelTag
	caCert         string
	servingInfo    *StateServingInfo
	apiDetails     *apiDetails
}

// ReadConfig reads configuration data from the given location.
func ReadConfig(configFilePath string) (Config, error) {
	var config *configInternal
	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot read agent config %q", configFilePath)
	}
	_, config, err = parseConfigData(configData)
	if err != nil {
		return nil, err
	}
	config.configFilePath = configFilePath
	return config, nil
}

func (c *configInternal) DataDir() string {
	return c.paths.DataDir
}

func (c *configInternal) LogDir() string {
	return c.paths.LogDir
}

func (c *configInternal) CACert() string {
	return c.caCert
}

func (c *configInternal) StateServingInfo() (StateServingInfo, bool) {
	if c.servingInfo == nil {
		return StateServingInfo{}, false
	}
	return *c.servingInfo, true
}

func (c *configInternal) APIAddresses() ([]string, error) {
	if c.apiDetails == nil {
		return []string{}, errors.New("No apidetails in config")
	}
	return append([]string{}, c.apiDetails.addresses...), nil
}

func (c *configInternal) Tag() names.Tag {
	return c.tag
}

func (c *configInternal) Model() names.ModelTag {
	return c.model
}

func (c *configInternal) Controller() names.ControllerTag {
	return c.controller
}

func (c *configInternal) Dir() string {
	return Dir(c.paths.DataDir, c.tag)
}
