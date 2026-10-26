package whatsmeow

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type outgoingMsgEntry struct {
	msg *waE2E.Message
	at  time.Time
	ttl time.Duration
}

const outgoingMsgCacheTTL = 30 * time.Minute

// statusMsgCacheTTL covers a status's whole life: a viewer whose device could not decrypt it asks
// for a retry whenever they open it, up to 24h later. Past 30 minutes the retry failed and the
// viewer was left on "waiting for this message".
const statusMsgCacheTTL = 24 * time.Hour

// CacheOutgoingMessage stores the raw protobuf of a sent message to allow
// reconstruction on retry receipts (including media, which cannot be rebuilt from the database).
func (m *Manager) CacheOutgoingMessage(id string, chat types.JID, msg *waE2E.Message) {
	if id == "" || msg == nil {
		return
	}
	now := time.Now()
	ttl := outgoingMsgCacheTTL
	if chat == types.StatusBroadcastJID {
		ttl = statusMsgCacheTTL
	}
	m.outgoingMsgCache.Store(id, outgoingMsgEntry{msg: msg, at: now, ttl: ttl})
	// Opportunistic cleanup of expired entries.
	m.outgoingMsgCache.Range(func(k, v any) bool {
		if e, ok := v.(outgoingMsgEntry); ok && now.Sub(e.at) > e.ttl {
			m.outgoingMsgCache.Delete(k)
		}
		return true
	})
}

func (m *Manager) getMessageForRetryCallback(instanceID string) func(types.JID, types.JID, string) *waE2E.Message {
	return func(chat, sender types.JID, id string) *waE2E.Message {
		m.log.Debug("WhatsMeow solicitou mensagem para retry",
			zap.String("instance_id", instanceID),
			zap.String("msg_id", id),
			zap.String("chat", chat.String()))

		// 1. In-memory cache of the raw protobuf — covers any type, including media.
		if val, ok := m.outgoingMsgCache.Load(id); ok {
			if entry, ok := val.(outgoingMsgEntry); ok && entry.msg != nil && time.Since(entry.at) <= entry.ttl {
				m.log.Info("mensagem reconstruída para retry via cache de protobuf",
					zap.String("instance_id", instanceID),
					zap.String("msg_id", id))
				return entry.msg
			}
			m.outgoingMsgCache.Delete(id)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		msg, err := m.messageRepo.GetByWhatsAppID(ctx, id)
		if err != nil {
			m.log.Warn("MENSAGEM NÃO ENCONTRADA no banco para retry solicitado pelo WhatsApp",
				zap.String("instance_id", instanceID),
				zap.String("msg_id", id),
				zap.String("chat", chat.String()),
				zap.Error(err))
			return nil
		}

		waMsg := &waE2E.Message{}
		switch msg.Type {
		case "text":
			waMsg.Conversation = proto.String(msg.Payload)
			m.log.Info("mensagem de TEXTO reconstruída para retry com sucesso",
				zap.String("instance_id", instanceID),
				zap.String("msg_id", id))
			return waMsg
		case "image", "video", "audio", "document":
			m.log.Warn("não foi possível reconstruir mensagem de MÍDIA para retry sem Raw protobuf (futura implementação)",
				zap.String("instance_id", instanceID),
				zap.String("msg_id", id),
				zap.String("type", msg.Type))
		default:
			m.log.Warn("tipo de mensagem desconhecido para reconstrução de retry",
				zap.String("type", msg.Type),
				zap.String("msg_id", id))
		}

		return nil
	}
}
