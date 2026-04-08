package domain

// Order status and pay status rules aligned with Java Orders constants.
const (
	OrderStatusPendingPayment   = 1
	OrderStatusToBeConfirmed    = 2
	OrderStatusConfirmed        = 3
	OrderStatusDeliveryProgress = 4
	OrderStatusCompleted        = 5
	OrderStatusCancelled        = 6

	PayStatusUnpaid = 0
	PayStatusPaid   = 1
	PayStatusRefund = 2
)

// Valid order in report metrics follows Java report/workspace SQL: COMPLETED only.
func IsValidOrderStatus(status int) bool {
	return status == OrderStatusCompleted
}

// Refund amount follows pay status = REFUND.
func IsRefundPayStatus(payStatus int) bool {
	return payStatus == PayStatusRefund
}
