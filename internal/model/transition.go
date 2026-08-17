package model

func CanTransition(from, to RequestStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusQueued:
		return to == StatusAssigned || to == StatusRejected
	case StatusAssigned:
		return to == StatusPending || to == StatusResolved || to == StatusRejected
	case StatusPending:
		return to == StatusAssigned || to == StatusResolved || to == StatusRejected
	case StatusResolved, StatusRejected:
		return false
	default:
		return false
	}
}

func AllowedStatuses() []RequestStatus {
	return []RequestStatus{StatusQueued, StatusAssigned, StatusPending, StatusResolved, StatusRejected}
}

func NormalizeStatus(value string) RequestStatus {
	status := RequestStatus(value)
	for _, candidate := range AllowedStatuses() {
		if candidate == status {
			return status
		}
	}
	return StatusQueued
}
