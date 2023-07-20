// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package agent

import (
	"github.com/juju/errors"
	"github.com/juju/names/v4"
	goyaml "gopkg.in/yaml.v2"
)

var format_2_0 = formatter_2_0{}

// formatter_2_0 is the formatter for the 2.0 format.
type formatter_2_0 struct {
}

// Ensure that the formatter_2_0 struct implements the formatter interface.
var _ formatter = formatter_2_0{}

// format_2_0Serialization holds information for a given agent.
type format_2_0Serialization struct {
	Tag     string `yaml:"tag,omitempty"`
	DataDir string `yaml:"datadir,omitempty"`
	LogDir  string `yaml:"logdir,omitempty"`

	CACert string `yaml:"cacert,omitempty"`

	Controller   string   `yaml:"controller,omitempty"`
	Model        string   `yaml:"model,omitempty"`
	APIAddresses []string `yaml:"apiaddresses,omitempty"`
	APIPassword  string   `yaml:"apipassword,omitempty"`

	// Only controller machines have these next items set.
	ControllerCert    string `yaml:"controllercert,omitempty"`
	ControllerKey     string `yaml:"controllerkey,omitempty"`
	CAPrivateKey      string `yaml:"caprivatekey,omitempty"`
	APIPort           int    `yaml:"apiport,omitempty"`
	ControllerAPIPort int    `yaml:"controllerapiport,omitempty"`
	SharedSecret      string `yaml:"sharedsecret,omitempty"`
	SystemIdentity    string `yaml:"systemidentity,omitempty"`
}

func init() {
	registerFormat(format_2_0)
}

func (formatter_2_0) version() string {
	return "2.0"
}

func (formatter_2_0) unmarshal(data []byte) (*configInternal, error) {
	// NOTE: this needs to handle the absence of StatePort and get it from the
	// address
	var format format_2_0Serialization
	if err := goyaml.Unmarshal(data, &format); err != nil {
		return nil, err
	}
	tag, err := names.ParseTag(format.Tag)
	if err != nil {
		return nil, err
	}
	controllerTag, err := names.ParseControllerTag(format.Controller)
	if err != nil {
		return nil, errors.Trace(err)
	}
	modelTag, err := names.ParseModelTag(format.Model)
	if err != nil {
		return nil, errors.Trace(err)
	}
	config := &configInternal{
		tag: tag,
		paths: NewPathsWithDefaults(Paths{
			DataDir: format.DataDir,
			LogDir:  format.LogDir,
		}),
		controller: controllerTag,
		model:      modelTag,
		caCert:     format.CACert,
	}
	if len(format.APIAddresses) > 0 {
		config.apiDetails = &apiDetails{
			addresses: format.APIAddresses,
		}
	}
	if len(format.ControllerKey) != 0 {
		config.servingInfo = &StateServingInfo{
			Cert:              format.ControllerCert,
			PrivateKey:        format.ControllerKey,
			CAPrivateKey:      format.CAPrivateKey,
			APIPort:           format.APIPort,
			ControllerAPIPort: format.ControllerAPIPort,
			SharedSecret:      format.SharedSecret,
			SystemIdentity:    format.SystemIdentity,
		}

	}
	return config, nil
}
