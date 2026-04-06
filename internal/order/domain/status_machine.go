package domain

var allowedTransitions = map[OrderStatus]map[OrderStatus]struct{}{
	OrderStatusPendingPay: {
		OrderStatusPaid:     {},
		OrderStatusCanceled: {},
	},
	OrderStatusPaid: {
		OrderStatusAccepted: {},
		OrderStatusCanceled: {},
	},
	OrderStatusAccepted: {
		OrderStatusDelivering: {},
		OrderStatusCanceled:   {},
	},
	OrderStatusDelivering: {
		OrderStatusCompleted: {},
	},
}

func CanCancel(status OrderStatus) bool {
	switch status {
	case OrderStatusPendingPay, OrderStatusPaid, OrderStatusAccepted:
		return true
	default:
		return false
	}
}

func CanTransit(from, to OrderStatus) bool {
	if from == to {
		return true
	}
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	_, exists := next[to]
	return exists
}
