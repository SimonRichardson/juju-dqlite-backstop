// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package net

import (
	"fmt"
	"net"

	"github.com/juju/collections/set"
	"github.com/juju/errors"
)

// ExternalIPs returns a list of non-loopback IP addresses
func ExternalIPs() (set.Strings, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	addresses := set.NewStrings()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			addresses.Add(ip.String())
		}
	}
	if addresses.Size() == 0 {
		return nil, fmt.Errorf("ip addresses %w", errors.NotFound)
	}
	return addresses, nil
}
