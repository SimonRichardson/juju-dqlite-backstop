// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package database

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/juju/collections/transform"
	"github.com/juju/errors"
	"gopkg.in/yaml.v3"

	"github.com/SimonRichardson/juju-dqlite-backstop/internal/agent"
	"github.com/SimonRichardson/juju-dqlite-backstop/internal/database/app"
	"github.com/SimonRichardson/juju-dqlite-backstop/internal/database/client"
	"github.com/SimonRichardson/juju-dqlite-backstop/internal/database/dqlite"
)

const (
	dqliteBootstrapBindIP = "127.0.0.1"
	dqliteDataDir         = "dqlite"
	dqlitePort            = 17666
	dqliteClusterFileName = "cluster.yaml"
)

// NodeManager is responsible for interrogating a single Dqlite node,
// and emitting configuration for starting its Dqlite `App` based on
// operational requirements and controller agent config.
type NodeManager struct {
	cfg    agent.Config
	port   int
	logger Logger

	dataDir string
}

// NewNodeManager returns a new NodeManager reference
// based on the input agent configuration.
func NewNodeManager(cfg agent.Config, logger Logger) *NodeManager {
	return &NodeManager{
		cfg:    cfg,
		port:   dqlitePort,
		logger: logger,
	}
}

// IsBootstrappedNode returns true if this machine or container was where we
// first bootstrapped Dqlite, and it hasn't been reconfigured since.
// Specifically, whether we are a cluster of one, and bound to the loopback
// IP address.
func (m *NodeManager) IsBootstrappedNode(ctx context.Context) (bool, error) {
	extant, err := m.IsExistingNode()
	if err != nil {
		return false, errors.Annotate(err, "determining existing Dqlite node")
	}
	if !extant {
		return false, nil
	}

	servers, err := m.ClusterServers(ctx)
	if err != nil {
		return false, errors.Trace(err)
	}

	if len(servers) != 1 {
		return false, nil
	}

	return strings.HasPrefix(servers[0].Address, dqliteBootstrapBindIP), nil
}

// IsExistingNode returns true if this machine or container has
// ever started a Dqlite `App` before. Specifically, this is whether
// the Dqlite data directory is empty.
func (m *NodeManager) IsExistingNode() (bool, error) {
	if _, err := m.EnsureDataDir(); err != nil {
		return false, errors.Annotate(err, "ensuring Dqlite data directory")
	}

	dir, err := os.Open(m.dataDir)
	if err != nil {
		return false, errors.Annotate(err, "opening Dqlite data directory")
	}

	_, err = dir.Readdirnames(1)
	switch err {
	case nil:
		return true, nil
	case io.EOF:
		return false, nil
	default:
		return false, errors.Annotate(err, "reading Dqlite data directory")
	}
}

// EnsureDataDir ensures that a directory for Dqlite data exists at
// a path determined by the agent config, then returns that path.
func (m *NodeManager) EnsureDataDir() (string, error) {
	if m.dataDir == "" {
		dir := filepath.Join(m.cfg.DataDir(), dqliteDataDir)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", errors.Annotatef(err, "creating directory for Dqlite data")
		}
		m.dataDir = dir
	}
	return m.dataDir, nil
}

// ClusterServers returns the node information for
// Dqlite nodes configured to be in the cluster.
func (m *NodeManager) ClusterServers(ctx context.Context) ([]dqlite.NodeInfo, error) {
	store, err := m.nodeClusterStore()
	if err != nil {
		return nil, errors.Trace(err)
	}
	servers, err := store.Get(ctx)
	return servers, errors.Annotate(err, "retrieving servers from Dqlite node store")
}

// SetClusterServers reconfigures the Dqlite cluster by writing the
// input servers to Dqlite's Raft log and the local node YAML store.
// This should only be called on a stopped Dqlite node.
func (m *NodeManager) SetClusterServers(ctx context.Context, servers []dqlite.NodeInfo) error {
	store, err := m.nodeClusterStore()
	if err != nil {
		return errors.Trace(err)
	}

	if err := dqlite.ReconfigureMembership(m.dataDir, servers); err != nil {
		return errors.Annotate(err, "reconfiguring Dqlite cluster membership")
	}

	return errors.Annotate(store.Set(ctx, servers), "writing servers to Dqlite node store")
}

// SetNodeInfo rewrites the local node information file in the Dqlite
// data directory, so that it matches the input NodeInfo.
// This should only be called on a stopped Dqlite node.
func (m *NodeManager) SetNodeInfo(server dqlite.NodeInfo) error {
	data, err := yaml.Marshal(server)
	if err != nil {
		return errors.Annotatef(err, "marshalling NodeInfo %#v", server)
	}
	return errors.Annotatef(
		os.WriteFile(path.Join(m.dataDir, "info.yaml"), data, 0600), "writing info.yaml to %s", m.dataDir)
}

// WithLoopbackAddressOption returns a Dqlite application
// Option that will bind Dqlite to the loopback IP.
func (m *NodeManager) WithLoopbackAddressOption() app.Option {
	return m.WithAddressOption(dqliteBootstrapBindIP)
}

// WithAddressOption returns a Dqlite application Option
// for specifying the local address:port to use.
func (m *NodeManager) WithAddressOption(ip string) app.Option {
	return app.WithAddress(fmt.Sprintf("%s:%d", ip, m.port))
}

// WithTLSOption returns a Dqlite application Option for TLS encryption
// of traffic between clients and clustered application nodes.
func (m *NodeManager) WithTLSOption() (app.Option, error) {
	stateInfo, ok := m.cfg.StateServingInfo()
	if !ok {
		return nil, errors.NotSupportedf("Dqlite node initialisation on non-controller machine/container")
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(m.cfg.CACert()))

	controllerCert, err := tls.X509KeyPair([]byte(stateInfo.Cert), []byte(stateInfo.PrivateKey))
	if err != nil {
		return nil, errors.Annotate(err, "parsing controller certificate")
	}

	listen := &tls.Config{
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{controllerCert},
	}

	dial := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{controllerCert},
		// We cannot provide a ServerName value here, so we rely on the
		// server validating the controller's client certificate.
		InsecureSkipVerify: true,
	}

	return app.WithTLS(listen, dial), nil
}

// WithClusterOption returns a Dqlite application Option for initialising
// Dqlite as the member of a cluster with peers representing other controllers.
func (m *NodeManager) WithClusterOption(addrs []string) app.Option {
	peerAddrs := transform.Slice(addrs, func(addr string) string {
		return fmt.Sprintf("%s:%d", addr, m.port)
	})

	m.logger.Debugf("determined Dqlite cluster members: %v", peerAddrs)
	return app.WithCluster(peerAddrs)
}

// nodeClusterStore returns a YamlNodeStore instance based
// on the cluster.yaml file in the Dqlite data directory.
func (m *NodeManager) nodeClusterStore() (*client.YamlNodeStore, error) {
	store, err := client.NewYamlNodeStore(path.Join(m.dataDir, dqliteClusterFileName))
	return store, errors.Annotate(err, "opening Dqlite cluster node store")
}
