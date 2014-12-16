package scheduler

import "github.com/citadel/citadel"

// PortScheduler will refuse to schedule a container where bound Ports conflict with other instances
type PortScheduler struct{}

func (s *PortScheduler) Schedule(i *citadel.Image, e *citadel.Engine) (bool, error) {
	containers, err := e.ListContainers(false)
	if err != nil {
		return false, err
	}

	if s.hasConflictingPorts(i.BindPorts, containers) {
		return false, nil
	}

	return true, nil
}

func (s *PortScheduler) hasConflictingPorts(imagePorts []*citadel.Port, containers []*citadel.Container) bool {
	for _, ct := range containers {
		for _, hostPort := range ct.Ports {
			for _, imagePort := range imagePorts {
				if hostPort.Port == imagePort.Port {
					return true
				}
			}
		}
	}

	return false
}
