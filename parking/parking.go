package parking

import (
	"fmt"
	"sync"

	"smart-parking-kv/kv"
)

type ParkingLot struct {
	store *kv.Store
	mu    sync.Mutex
}

func NewParkingLot() *ParkingLot {
	return &ParkingLot{
		store: kv.NewStore(nil),
	}
}

func NewParkingLotWithStore(store *kv.Store) *ParkingLot {
	return &ParkingLot{
		store: store,
	}
}

// Enter parks a vehicle in a slot.
func (p *ParkingLot) Enter(slotID string, vehicleID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	status, exists := p.store.Get("slot:" + slotID)

	if exists && status == "OCCUPIED" {
		return fmt.Errorf("slot %s is already occupied", slotID)
	}

	err := p.store.Put("slot:"+slotID, "OCCUPIED")
	if err != nil {
		return fmt.Errorf("failed to update slot: %w", err)
	}

	err = p.store.Put("vehicle:"+vehicleID, slotID)
	if err != nil {
		return fmt.Errorf("failed to store vehicle: %w", err)
	}

	return nil
}

// Exit removes a vehicle from a slot.
func (p *ParkingLot) Exit(slotID string, vehicleID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	status, exists := p.store.Get("slot:" + slotID)

	if !exists || status != "OCCUPIED" {
		return fmt.Errorf("slot %s is not occupied", slotID)
	}

	parkedSlot, exists := p.store.Get("vehicle:" + vehicleID)
	if !exists || parkedSlot != slotID {
		return fmt.Errorf("vehicle %s is not parked in slot %s", vehicleID, slotID)
	}

	err := p.store.Put("slot:"+slotID, "AVAILABLE")
	if err != nil {
		return fmt.Errorf("failed to update slot: %w", err)
	}

	err = p.store.Delete("vehicle:" + vehicleID)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle: %w", err)
	}

	return nil
}

// Status returns the current status of a slot.
func (p *ParkingLot) Status(slotID string) string {
	status, exists := p.store.Get("slot:" + slotID)

	if !exists {
		return "AVAILABLE"
	}

	return status
}

// Available returns the number of available slots.
func (p *ParkingLot) Available(slotIDs []string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	count := 0

	for _, slotID := range slotIDs {
		status, exists := p.store.Get("slot:" + slotID)

		if !exists || status == "AVAILABLE" {
			count++
		}
	}

	return count
}
