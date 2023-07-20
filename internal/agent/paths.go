// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package agent

type OS int // strongly typed runtime.GOOS value to help with refactoring

const (
	OSUnixLike OS = 1
)

type osVarType int

const (
	tmpDir osVarType = iota
	logDir
	dataDir
	confDir
	certDir
)

const (
	// NixDataDir is location for agent binaries on *nix operating systems.
	NixDataDir = "/var/lib/juju"

	// NixLogDir is location for Juju logs on *nix operating systems.
	NixLogDir = "/var/log"
)

var nixVals = map[osVarType]string{
	tmpDir:  "/tmp",
	logDir:  NixLogDir,
	dataDir: NixDataDir,
	confDir: "/etc/juju",
	certDir: "/etc/juju/certs.d",
}

// CurrentOS returns the OS value for the currently-running system.
func CurrentOS() OS {
	return OSUnixLike
}

// OSType converts the given os name to an OS value.
func OSType(osName string) OS {
	return OSUnixLike
}

// osVal will lookup the value of the key valname
// in the appropriate map, based on the OS value.
func osVal(os OS, valname osVarType) string {
	return nixVals[valname]
}

// LogDir returns filesystem path the directory where juju may
// save log files.
func LogDir(os OS) string {
	return osVal(os, logDir)
}

// DataDir returns a filesystem path to the folder used by juju to
// store tools, charms, locks, etc
func DataDir(os OS) string {
	return osVal(os, dataDir)
}

// CertDir returns a filesystem path to the folder used by juju to
// store certificates that are added by default to the Juju client
// api certificate pool.
func CertDir(os OS) string {
	return osVal(os, certDir)
}

// ConfDir returns the path to the directory where Juju may store
// configuration files.
func ConfDir(os OS) string {
	return osVal(os, confDir)
}
