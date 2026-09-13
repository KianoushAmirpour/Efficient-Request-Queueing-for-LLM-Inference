package domain

type CoalescingDecision struct {
	Action CoalescingAction
	Error  error
}

type CoalescingAction string

const (
	PROCEED  CoalescingAction = "PROCEED"
	REJECTED CoalescingAction = "REJECTED"
	UNKNOWN  CoalescingAction = "UNKNOWN"
)
