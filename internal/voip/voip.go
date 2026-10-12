// Package voip holds the call types shared by the session manager (which drives the
// engine) and the webhook layer (which turns them into call.* events), so neither
// has to import the other.
package voip

import (
	"errors"
	"time"
)

var (
	ErrCallsDisabled = errors.New("chamadas desativadas nesta instância")
	ErrCallNotFound  = errors.New("chamada não encontrada")
)

const (
	DirectionIncoming = "incoming"
	DirectionOutgoing = "outgoing"
)

// Call is the public snapshot of a call the engine is tracking.
type Call struct {
	ID        string    `json:"callId"`
	Peer      string    `json:"peer"`
	Direction string    `json:"direction"`
	State     string    `json:"state"`
	StartedAt time.Time `json:"startedAt"`
}

// StateEvent is emitted on every engine phase change. Consumers must treat call
// webhooks as idempotent per callId: a peer hangup arrives both from whatsmeow
// (call.ended) and from the engine (state "ended").
type StateEvent struct {
	Call
	Reason string `json:"reason,omitempty"`
}
