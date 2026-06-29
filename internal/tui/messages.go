package tui

import "github.com/d-init-d/d-research-cli/internal/event"

type busEventMsg struct {
	event event.Event
}

type runFinishedMsg struct {
	err          error
	awaitApprove bool
}

type approveFinishedMsg struct {
	err error
}

type doctorFinishedMsg struct {
	ok     bool
	status string
	detail string
}

type screenChangeMsg int