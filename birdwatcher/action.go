package birdwatcher

import "net"

type Action struct {
	Service  *ServiceCheck
	State    ServiceState
	Prefixes []net.IPNet
}
