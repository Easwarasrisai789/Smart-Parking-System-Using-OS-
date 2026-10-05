package parking

import (
	"fmt"
	"testing"
)

func TestParkingEnterAndExit(t *testing.T) {
	parking := NewParkingLot()

	// Vehicle enters the parking lot.
	err := parking.Enter("001", "AP39AB1234")

	if err != nil {
		t.Fatalf("enter failed: %v", err)
	}

	// Check that the slot is occupied.
	status := parking.Status("001")

	if status != "OCCUPIED" {
		t.Fatalf("expected OCCUPIED, got %s", status)
	}

	// Vehicle exits the parking lot.
	err = parking.Exit("001", "AP39AB1234")

	if err != nil {
		t.Fatalf("exit failed: %v", err)
	}

	// Check that the slot is available again.
	status = parking.Status("001")

	if status != "AVAILABLE" {
		t.Fatalf("expected AVAILABLE, got %s", status)
	}
}

func TestCannotEnterOccupiedSlot(t *testing.T) {
	parking := NewParkingLot()

	// First vehicle enters.
	err := parking.Enter("001", "AP39AB1234")

	if err != nil {
		t.Fatalf("first enter failed: %v", err)
	}

	// Second vehicle tries to use the same slot.
	err = parking.Enter("001", "AP40XY5678")

	if err == nil {
		t.Fatal("expected error when entering occupied slot")
	}
}

func TestCannotExitSlotWithDifferentVehicle(t *testing.T) {
	parking := NewParkingLot()

	if err := parking.Enter("001", "VEHICLE-1"); err != nil {
		t.Fatalf("enter failed: %v", err)
	}

	if err := parking.Exit("001", "VEHICLE-2"); err == nil {
		t.Fatal("expected exit to fail for a vehicle not parked in the slot")
	}
	if status := parking.Status("001"); status != "OCCUPIED" {
		t.Fatalf("slot status after wrong vehicle exit = %s, want OCCUPIED", status)
	}

	if err := parking.Exit("001", "VEHICLE-1"); err != nil {
		t.Fatalf("parked vehicle should be able to exit: %v", err)
	}
	if status := parking.Status("001"); status != "AVAILABLE" {
		t.Fatalf("slot status after correct vehicle exit = %s, want AVAILABLE", status)
	}
}

func TestAvailableSlots(t *testing.T) {
	parking := NewParkingLot()

	slots := []string{
		"001",
		"002",
		"003",
	}

	// Initially all 3 slots are available.
	count := parking.Available(slots)

	if count != 3 {
		t.Fatalf("expected 3 available slots, got %d", count)
	}

	// Occupy one slot.
	err := parking.Enter("001", "AP39AB1234")

	if err != nil {
		t.Fatalf("enter failed: %v", err)
	}

	// Now only 2 slots should be available.
	count = parking.Available(slots)

	if count != 2 {
		t.Fatalf("expected 2 available slots, got %d", count)
	}
}

func TestConcurrentParking(t *testing.T) {
	parking := NewParkingLot()

	slots := []string{
		"001",
		"002",
		"003",
		"004",
		"005",
	}

	done := make(chan bool)

	// Start multiple vehicles at the same time.
	for i, slot := range slots {
		go func(i int, slot string) {
			vehicleID := fmt.Sprintf("VEHICLE-%d", i)

			err := parking.Enter(slot, vehicleID)

			if err != nil {
				t.Errorf(
					"vehicle %s failed to enter: %v",
					vehicleID,
					err,
				)
			}

			done <- true
		}(i, slot)
	}

	// Wait for all vehicles.
	for range slots {
		<-done
	}

	// All 5 slots should be occupied.
	available := parking.Available(slots)

	if available != 0 {
		t.Fatalf("expected 0 available slots, got %d", available)
	}
}

func TestParkingStress(t *testing.T) {
	parking := NewParkingLot()

	const numberOfVehicles = 100

	slots := make([]string, numberOfVehicles)

	for i := 0; i < numberOfVehicles; i++ {
		slots[i] = fmt.Sprintf("%03d", i+1)
	}

	done := make(chan bool)

	// 100 vehicles enter at the same time.
	for i := 0; i < numberOfVehicles; i++ {
		go func(i int) {
			vehicleID := fmt.Sprintf("VEHICLE-%03d", i+1)

			err := parking.Enter(slots[i], vehicleID)

			if err != nil {
				t.Errorf(
					"vehicle %s failed to enter: %v",
					vehicleID,
					err,
				)
			}

			done <- true
		}(i)
	}

	// Wait for all vehicles to enter.
	for i := 0; i < numberOfVehicles; i++ {
		<-done
	}

	// All 100 slots should be occupied.
	available := parking.Available(slots)

	if available != 0 {
		t.Fatalf(
			"expected 0 available slots, got %d",
			available,
		)
	}

	// 100 vehicles exit at the same time.
	for i := 0; i < numberOfVehicles; i++ {
		go func(i int) {
			vehicleID := fmt.Sprintf("VEHICLE-%03d", i+1)

			err := parking.Exit(slots[i], vehicleID)

			if err != nil {
				t.Errorf(
					"vehicle %s failed to exit: %v",
					vehicleID,
					err,
				)
			}

			done <- true
		}(i)
	}

	// Wait for all vehicles to exit.
	for i := 0; i < numberOfVehicles; i++ {
		<-done
	}

	// All 100 slots should now be available.
	available = parking.Available(slots)

	if available != numberOfVehicles {
		t.Fatalf(
			"expected %d available slots, got %d",
			numberOfVehicles,
			available,
		)
	}
}

func TestRepeatedParkingStress(t *testing.T) {
	const (
		rounds        = 10
		numberOfSlots = 100
	)

	for round := 0; round < rounds; round++ {
		parking := NewParkingLot()

		slots := make([]string, numberOfSlots)

		for i := 0; i < numberOfSlots; i++ {
			slots[i] = fmt.Sprintf("%03d", i+1)
		}

		done := make(chan bool)

		// Concurrent ENTER operations.
		for i := 0; i < numberOfSlots; i++ {
			go func(i int) {
				vehicleID := fmt.Sprintf(
					"ROUND-%d-VEHICLE-%03d",
					round,
					i+1,
				)

				err := parking.Enter(slots[i], vehicleID)

				if err != nil {
					t.Errorf(
						"round %d: vehicle %s failed to enter: %v",
						round,
						vehicleID,
						err,
					)
				}

				done <- true
			}(i)
		}

		// Wait for all ENTER operations.
		for i := 0; i < numberOfSlots; i++ {
			<-done
		}

		// All slots should be occupied.
		available := parking.Available(slots)

		if available != 0 {
			t.Fatalf(
				"round %d: expected 0 available slots, got %d",
				round,
				available,
			)
		}

		// Concurrent EXIT operations.
		for i := 0; i < numberOfSlots; i++ {
			go func(i int) {
				vehicleID := fmt.Sprintf(
					"ROUND-%d-VEHICLE-%03d",
					round,
					i+1,
				)

				err := parking.Exit(slots[i], vehicleID)

				if err != nil {
					t.Errorf(
						"round %d: vehicle %s failed to exit: %v",
						round,
						vehicleID,
						err,
					)
				}

				done <- true
			}(i)
		}

		// Wait for all EXIT operations.
		for i := 0; i < numberOfSlots; i++ {
			<-done
		}

		// All slots should be available again.
		available = parking.Available(slots)

		if available != numberOfSlots {
			t.Fatalf(
				"round %d: expected %d available slots, got %d",
				round,
				numberOfSlots,
				available,
			)
		}
	}
}
