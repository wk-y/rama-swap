package main

import "sync"

type portManager struct {
	StartingPort      int
	reservedPortsLock sync.Mutex
	reservedPorts     map[int]struct{}
}

func newPortManager(startingPort int) *portManager {
	return &portManager{
		StartingPort:  startingPort,
		reservedPorts: map[int]struct{}{},
	}
}

func (p *portManager) ReservePort() int {
	p.reservedPortsLock.Lock()
	defer p.reservedPortsLock.Unlock()
	port := p.StartingPort
	for {
		if _, ok := p.reservedPorts[port]; !ok {
			p.reservedPorts[port] = struct{}{}
			return port
		}
	}
}

func (p *portManager) ReleasePort(port int) {
	p.reservedPortsLock.Lock()
	delete(p.reservedPorts, port)
	p.reservedPortsLock.Unlock()
}
