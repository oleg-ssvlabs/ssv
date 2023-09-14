package dutystorage

import (
	"sync"

	eth2apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"golang.org/x/exp/maps"
)

type Duty interface {
	eth2apiv1.AttesterDuty | eth2apiv1.ProposerDuty
}

type Duties[D Duty] struct {
	mu sync.RWMutex
	m  map[phase0.Epoch]map[phase0.Slot]map[phase0.ValidatorIndex]*D
}

func NewDuties[D Duty]() *Duties[D] {
	return &Duties[D]{
		m: make(map[phase0.Epoch]map[phase0.Slot]map[phase0.ValidatorIndex]*D),
	}
}

func (d *Duties[D]) SlotDuties(epoch phase0.Epoch, slot phase0.Slot) []*D {
	d.mu.RLock()
	defer d.mu.RUnlock()

	slotMap, ok := d.m[epoch]
	if !ok {
		return nil
	}

	duties, ok := slotMap[slot]
	if !ok {
		return nil
	}

	return maps.Values(duties)
}

func (d *Duties[D]) ValidatorDuty(epoch phase0.Epoch, slot phase0.Slot, validatorIndex phase0.ValidatorIndex) *D {
	d.mu.RLock()
	defer d.mu.RUnlock()

	slotMap, ok := d.m[epoch]
	if !ok {
		return nil
	}

	duties, ok := slotMap[slot]
	if !ok {
		return nil
	}

	duty, ok := duties[validatorIndex]
	if !ok {
		return nil
	}

	return duty
}

func (d *Duties[D]) Add(epoch phase0.Epoch, slot phase0.Slot, validatorIndex phase0.ValidatorIndex, duty *D) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.m[epoch]; !ok {
		d.m[epoch] = make(map[phase0.Slot]map[phase0.ValidatorIndex]*D)
	}
	if _, ok := d.m[epoch][slot]; !ok {
		d.m[epoch][slot] = make(map[phase0.ValidatorIndex]*D)
	}
	d.m[epoch][slot][validatorIndex] = duty
}

func (d *Duties[D]) ResetEpoch(epoch phase0.Epoch) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.m, epoch)
}
