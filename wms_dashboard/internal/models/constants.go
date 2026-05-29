package models

// Movement Types
const (
	MvtTypeInbound  = "INBOUND"
	MvtTypeOutbound = "OUTBOUND"
	MvtTypeRTV      = "RTV"
	MvtTypeTransfer = "TRANSFER"
	MvtTypeInternal = "INTERNAL"
)

// Movement Statuses
const (
	MvtStatusOpen       = "OPEN"
	MvtStatusInProgress = "IN_PROGRESS"
	MvtStatusInbound    = "INBOUND"
	MvtStatusReceipt    = "RECEIPT"
	MvtStatusJournaled  = "JOURNALED"
	MvtStatusCompleted  = "COMPLETED"
	MvtStatusRejected   = "REJECTED"
	MvtStatusShipping   = "SHIPPING"
	MvtStatusOutbound   = "OUTBOUND"
)

// HasReservation returns true if the movement type reserves stock upon creation
func HasReservation(mvtType string) bool {
	return mvtType == MvtTypeOutbound || mvtType == MvtTypeRTV || mvtType == MvtTypeTransfer
}
