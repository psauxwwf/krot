//go:build android
// +build android

package anet

import "net"

// Go 1.26 removed compatibility with anet's old zoneCache linkname hack.
// For Android API 24+ we only need standard library interface enumeration.
func Interfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

// InterfaceAddrs returns a list of the system's unicast interface addresses.
func InterfaceAddrs() ([]net.Addr, error) {
	return net.InterfaceAddrs()
}

// InterfaceAddrsByInterface returns addresses for a specific interface.
func InterfaceAddrsByInterface(ifi *net.Interface) ([]net.Addr, error) {
	return ifi.Addrs()
}

func SetAndroidVersion(version uint) {}
