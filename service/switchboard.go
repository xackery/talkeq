package service

import (
	"github.com/xackery/talkeq/model"
)

// Switchboard contains a list of patches
type Switchboard struct {
	Patches []*Patch
}

// Patch represents the link between two services
type Patch struct {
	Number string
	From   Service
	To     Service
}

// FindPatch iterates the patches and creates a list of valid ones
func (s *Switchboard) FindPatch(Number model.MessageType, from Service) (services []Service) {

	for _, patch := range s.Patches {
		if patch.Number != Number.String() {
			continue
		}
		if from != patch.From {
			continue
		}
		services = append(services, patch.To)
	}
	return
}
