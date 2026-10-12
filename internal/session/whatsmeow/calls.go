package whatsmeow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/purpshell/meowcaller"
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"

	"github.com/open-apime/apime/internal/voip"
)

// callEngine is the VoIP engine of one instance plus the calls it is tracking.
type callEngine struct {
	client *meowcaller.Client
	wa     *whatsmeow.Client
	mu     sync.Mutex
	calls  map[string]*trackedCall
}

type trackedCall struct {
	call      *meowcaller.Call
	direction string
	startedAt time.Time
}

// attachCalls installs the engine on a client that has not connected yet. It must run
// before Connect: meowcaller hooks whatsmeow's <ack> handler, which is only safe before
// the receive loop starts. Instances without calls_enabled are left untouched, since the
// engine preaccepts every incoming call it sees.
func (m *Manager) attachCalls(ctx context.Context, instanceID string, client *whatsmeow.Client) {
	m.detachCalls(instanceID)
	if m.instanceRepo == nil {
		return
	}
	inst, err := m.instanceRepo.GetByID(ctx, instanceID)
	if err != nil || !inst.CallsEnabled {
		return
	}

	logger := zerolog.New(zerologToZap{log: m.log, instanceID: instanceID}).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	eng := &callEngine{
		client: meowcaller.NewClient(client, meowcaller.WithLogger(logger)),
		wa:     client,
		calls:  make(map[string]*trackedCall),
	}
	eng.client.OnIncomingCall(func(call *meowcaller.Call) {
		m.trackCall(instanceID, eng, call, voip.DirectionIncoming)
	})

	m.voipMu.Lock()
	m.voip[instanceID] = eng
	m.voipMu.Unlock()
	m.log.Info("motor de chamadas instalado", zap.String("instance_id", instanceID))
}

func (m *Manager) detachCalls(instanceID string) {
	m.voipMu.Lock()
	eng := m.voip[instanceID]
	delete(m.voip, instanceID)
	m.voipMu.Unlock()
	if eng == nil {
		return
	}
	// Hang up outside the lock: OnEnd fires back into eng.mu.
	eng.mu.Lock()
	active := make([]*trackedCall, 0, len(eng.calls))
	for _, tc := range eng.calls {
		active = append(active, tc)
	}
	eng.mu.Unlock()
	for _, tc := range active {
		_ = tc.call.Hangup()
	}
}

func (m *Manager) callEngineFor(instanceID string) (*callEngine, error) {
	m.voipMu.Lock()
	defer m.voipMu.Unlock()
	eng := m.voip[instanceID]
	if eng == nil {
		return nil, voip.ErrCallsDisabled
	}
	return eng, nil
}

func (m *Manager) trackCall(instanceID string, eng *callEngine, call *meowcaller.Call, direction string) {
	tc := &trackedCall{call: call, direction: direction, startedAt: time.Now()}
	eng.mu.Lock()
	eng.calls[call.ID()] = tc
	eng.mu.Unlock()

	call.OnStateChange(func(phase meowcaller.CallPhase) {
		m.emitCallState(instanceID, eng, tc, phaseName(phase), "")
	})
	call.OnEnd(func(reason string) {
		eng.mu.Lock()
		delete(eng.calls, call.ID())
		eng.mu.Unlock()
		m.emitCallState(instanceID, eng, tc, "ended", reason)
	})
	m.emitCallState(instanceID, eng, tc, phaseName(call.State()), "")
}

func (m *Manager) emitCallState(instanceID string, eng *callEngine, tc *trackedCall, state, reason string) {
	m.mu.RLock()
	handler := m.eventHandler
	m.mu.RUnlock()
	if handler == nil {
		return
	}
	jid := ""
	if eng.wa.Store != nil && eng.wa.Store.ID != nil {
		jid = eng.wa.Store.ID.String()
	}
	evt := &voip.StateEvent{Call: snapshot(tc, state), Reason: reason}
	go handler.Handle(context.Background(), instanceID, jid, eng.wa, evt)
}

func snapshot(tc *trackedCall, state string) voip.Call {
	return voip.Call{
		ID:        tc.call.ID(),
		Peer:      tc.call.Peer().String(),
		Direction: tc.direction,
		State:     state,
		StartedAt: tc.startedAt,
	}
}

func phaseName(p meowcaller.CallPhase) string {
	switch p {
	case meowcaller.CallPhaseCalling:
		return "calling"
	case meowcaller.CallPhaseRinging:
		return "ringing"
	case meowcaller.CallPhaseConnecting:
		return "connecting"
	case meowcaller.CallPhaseActive:
		return "active"
	case meowcaller.CallPhaseEnded:
		return "ended"
	default:
		return "idle"
	}
}

func (m *Manager) StartCall(ctx context.Context, instanceID, to string) (voip.Call, error) {
	eng, err := m.callEngineFor(instanceID)
	if err != nil {
		return voip.Call{}, err
	}
	call, err := eng.client.Call(ctx, to)
	if err != nil {
		return voip.Call{}, err
	}
	m.trackCall(instanceID, eng, call, voip.DirectionOutgoing)
	return m.GetCall(instanceID, call.ID())
}

func (m *Manager) GetCall(instanceID, callID string) (voip.Call, error) {
	tc, err := m.lookupCall(instanceID, callID)
	if err != nil {
		return voip.Call{}, err
	}
	return snapshot(tc, phaseName(tc.call.State())), nil
}

func (m *Manager) ListCalls(instanceID string) ([]voip.Call, error) {
	eng, err := m.callEngineFor(instanceID)
	if err != nil {
		return nil, err
	}
	eng.mu.Lock()
	defer eng.mu.Unlock()
	out := make([]voip.Call, 0, len(eng.calls))
	for _, tc := range eng.calls {
		out = append(out, snapshot(tc, phaseName(tc.call.State())))
	}
	return out, nil
}

func (m *Manager) AnswerCall(instanceID, callID string) error {
	tc, err := m.lookupCall(instanceID, callID)
	if err != nil {
		return err
	}
	return tc.call.Answer()
}

func (m *Manager) HangupCall(instanceID, callID string) error {
	tc, err := m.lookupCall(instanceID, callID)
	if err != nil {
		return err
	}
	return tc.call.Hangup()
}

// RejectCall declines an incoming call. It works without the engine too, through
// whatsmeow's own reject, which is what an auto-reject on a calls-disabled instance uses;
// that path needs the caller JID from the call.offer webhook.
func (m *Manager) RejectCall(ctx context.Context, instanceID, callID, from string) error {
	if tc, err := m.lookupCall(instanceID, callID); err == nil {
		return tc.call.Reject()
	}
	if strings.TrimSpace(from) == "" {
		return voip.ErrCallNotFound
	}
	jid, err := types.ParseJID(from)
	if err != nil {
		return fmt.Errorf("from inválido: %w", err)
	}
	client, err := m.GetClient(instanceID)
	if err != nil {
		return err
	}
	return client.RejectCall(ctx, jid, callID)
}

func (m *Manager) lookupCall(instanceID, callID string) (*trackedCall, error) {
	eng, err := m.callEngineFor(instanceID)
	if err != nil {
		return nil, err
	}
	eng.mu.Lock()
	defer eng.mu.Unlock()
	tc := eng.calls[callID]
	if tc == nil {
		return nil, voip.ErrCallNotFound
	}
	return tc, nil
}

// ReloadClient swaps the instance's client for a fresh one, so a calls_enabled change
// takes effect: the engine can only be installed before Connect. The old client's
// handlers are removed first, so its Disconnected never reaches the debounce and the
// consumer does not see the connection drop.
func (m *Manager) ReloadClient(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	old, ok := m.clients[instanceID]
	if ok {
		delete(m.clients, instanceID)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}

	old.RemoveEventHandlers()
	old.Disconnect()
	m.detachCalls(instanceID)

	_, err := m.restoreSessionIfExists(ctx, instanceID)
	return err
}

// zerologToZap forwards meowcaller's zerolog lines into the apime zap log.
type zerologToZap struct {
	log        *zap.Logger
	instanceID string
}

func (w zerologToZap) Write(p []byte) (int, error) {
	w.log.Info("meowcaller", zap.String("instance_id", w.instanceID), zap.ByteString("event", p))
	return len(p), nil
}
