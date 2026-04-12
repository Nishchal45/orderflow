package model

import "testing"

func TestCanTransitionTo_ValidTransitions(t *testing.T) {
	tests := []struct {
		name   string
		from   OrderStatus
		to     OrderStatus
		expect bool
	}{
		// Happy path transitions
		{"CREATED → INVENTORY_RESERVING", StatusCreated, StatusInventoryReserving, true},
		{"CREATED → PAYMENT_PENDING", StatusCreated, StatusPaymentPending, true},
		{"CREATED → REJECTED", StatusCreated, StatusRejected, true},
		{"INVENTORY_RESERVING → PAYMENT_PENDING", StatusInventoryReserving, StatusPaymentPending, true},
		{"INVENTORY_RESERVING → REJECTED", StatusInventoryReserving, StatusRejected, true},
		{"PAYMENT_PENDING → CONFIRMED", StatusPaymentPending, StatusConfirmed, true},
		{"PAYMENT_PENDING → CANCELLED", StatusPaymentPending, StatusCancelled, true},
		{"PAYMENT_PENDING → ROLLING_BACK", StatusPaymentPending, StatusRollingBack, true},
		{"ROLLING_BACK → CANCELLED", StatusRollingBack, StatusCancelled, true},
		{"CONFIRMED → SHIPPED", StatusConfirmed, StatusShipped, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.from.CanTransitionTo(tt.to)
			if got != tt.expect {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v", tt.from, tt.to, got, tt.expect)
			}
		})
	}
}

func TestCanTransitionTo_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from OrderStatus
		to   OrderStatus
	}{
		// Can't skip steps
		{"CREATED → CONFIRMED (skip steps)", StatusCreated, StatusConfirmed},
		{"CREATED → SHIPPED (skip steps)", StatusCreated, StatusShipped},
		{"CREATED → CANCELLED (no direct cancel)", StatusCreated, StatusCancelled},

		// Can't go backwards
		{"CONFIRMED → CREATED (backward)", StatusConfirmed, StatusCreated},
		{"CONFIRMED → PAYMENT_PENDING (backward)", StatusConfirmed, StatusPaymentPending},
		{"SHIPPED → CONFIRMED (backward)", StatusShipped, StatusConfirmed},

		// Terminal states can't transition
		{"SHIPPED → anything", StatusShipped, StatusConfirmed},
		{"CANCELLED → anything", StatusCancelled, StatusCreated},
		{"REJECTED → anything", StatusRejected, StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.from.CanTransitionTo(tt.to)
			if got {
				t.Errorf("CanTransitionTo(%s → %s) = true, want false (invalid transition)", tt.from, tt.to)
			}
		})
	}
}

func TestCanTransitionTo_UnknownStatus(t *testing.T) {
	unknown := OrderStatus("UNKNOWN")
	if unknown.CanTransitionTo(StatusCreated) {
		t.Error("unknown status should not transition to anything")
	}
}
