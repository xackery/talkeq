package service

// Switchboard contains a list of patches
type Switchboard struct {
	Patches []*Patch
}

// Patch represents the link between two services
type Patch struct {
	ChanNum int32
	From    Service
	To      Service
}

// FindPatch iterates the patches and creates a list of valid ones
func (s *Switchboard) FindPatch(chanNum int32, from Service) (services []Service) {

	for _, patch := range s.Patches {
		if patch.ChanNum != chanNum {
			continue
		}
		if from != patch.From {
			continue
		}
		services = append(services, patch.To)
	}
	return
}
