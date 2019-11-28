package birdwatcher

import "net"

// Action reflects the change to a specific state for a service and its prefixes
type Action struct {
	Service  *ServiceCheck
	State    ServiceState
	Prefixes []net.IPNet
}
