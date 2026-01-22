package message

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"mime"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/open-apime/apime/internal/storage"
	"github.com/open-apime/apime/internal/storage/model"
)

var (
	ErrInvalidPayload       = errors.New("payload inválido")
	ErrInstanceNotConnected = errors.New("instância não conectada")
	ErrInvalidJID           = errors.New("JID inválido")
	ErrUnsupportedMediaType = errors.New("tipo de mídia não suportado")
)

type Service struct {
	repo         storage.MessageRepository
	sessionMgr   SessionManager
	instanceRepo storage.InstanceRepository
	log          *zap.Logger
}

type SessionManager interface {
	GetClient(instanceID string) (*whatsmeow.Client, error)
	IsSessionReady(instanceID string) bool
	HasSession(instanceID string, jid types.JID) (bool, error)
}

func NewService(repo storage.MessageRepository, log *zap.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

func NewServiceWithSession(repo storage.MessageRepository, sessionMgr SessionManager, instanceRepo storage.InstanceRepository, log *zap.Logger) *Service {
	return &Service{
		repo:         repo,
		sessionMgr:   sessionMgr,
		instanceRepo: instanceRepo,
		log:          log,
	}
}

type EnqueueInput struct {
	InstanceID string
	To         string
	Type       string
	Payload    string
}

func (s *Service) Enqueue(ctx context.Context, input EnqueueInput) (model.Message, error) {
	if input.InstanceID == "" || input.To == "" || input.Payload == "" {
		return model.Message{}, ErrInvalidPayload
	}
	message := model.Message{
		ID:         uuid.NewString(),
		InstanceID: input.InstanceID,
		To:         input.To,
		Type:       input.Type,
		Payload:    input.Payload,
		Status:     "queued",
	}
	return s.repo.Create(ctx, message)
}

type SendInput struct {
	InstanceID string
	To         string
	Type       string
	Text       string
	MediaData  []byte
	MediaType  string
	Caption    string
	FileName   string
	Seconds    int
	PTT        bool
}

func (s *Service) Send(ctx context.Context, input SendInput) (model.Message, error) {
	if s.sessionMgr == nil {
		return model.Message{}, errors.New("session manager não configurado")
	}

	if input.InstanceID == "" || input.To == "" {
		return model.Message{}, ErrInvalidPayload
	}

	instance, err := s.instanceRepo.GetByID(ctx, input.InstanceID)
	if err != nil {
		return model.Message{}, fmt.Errorf("instância não encontrada: %w", err)
	}

	if instance.Status != model.InstanceStatusActive {
		return model.Message{}, ErrInstanceNotConnected
	}

	client, err := s.sessionMgr.GetClient(input.InstanceID)
	if err != nil {
		ctxUpdate := context.Background()
		if instToUpdate, fetchErr := s.instanceRepo.GetByID(ctxUpdate, input.InstanceID); fetchErr == nil {
			instToUpdate.Status = model.InstanceStatusError
			_, _ = s.instanceRepo.Update(ctxUpdate, instToUpdate)
		}
		return model.Message{}, fmt.Errorf("cliente não encontrado: %w", err)
	}

	if !client.IsLoggedIn() {
		ctxUpdate := context.Background()
		if instToUpdate, fetchErr := s.instanceRepo.GetByID(ctxUpdate, input.InstanceID); fetchErr == nil {
			instToUpdate.Status = model.InstanceStatusError
			_, _ = s.instanceRepo.Update(ctxUpdate, instToUpdate)
		}
		return model.Message{}, ErrInstanceNotConnected
	}

	readyStart := time.Now()
	isReady := false
	poked := false

	for time.Since(readyStart) < 10*time.Second {
		if s.sessionMgr.IsSessionReady(input.InstanceID) {
			isReady = true
			break
		}

		if time.Since(readyStart) > 2*time.Second && !poked {
			_ = client.SendPresence(ctx, types.PresenceAvailable)
			poked = true
		}

		time.Sleep(500 * time.Millisecond)
	}

	if !isReady {
		return model.Message{}, fmt.Errorf("sessão indisponível para criptografia, tente novamente em instantes")
	}

	toJID, err := s.resolveJID(ctx, client, input.To)
	if err != nil {
		return model.Message{}, fmt.Errorf("%w: %s", ErrInvalidJID, input.To)
	}

	if toJID.Server == types.DefaultUserServer || toJID.Server == types.HiddenUserServer {
		hasSession, err := s.sessionMgr.HasSession(input.InstanceID, toJID)

		_ = client.SendPresence(ctx, types.PresenceAvailable)

		_ = client.SendChatPresence(ctx, toJID, types.ChatPresenceComposing, types.ChatPresenceMediaText)

		if err == nil && !hasSession {
			s.log.Info("Nova sessão detectada...",
				zap.String("instance_id", input.InstanceID),
				zap.String("to", toJID.String()))

			wait := 1500 + rand.Intn(1500)
			time.Sleep(time.Duration(wait) * time.Millisecond)
		} else {
			wait := 500 + rand.Intn(700)
			time.Sleep(time.Duration(wait) * time.Millisecond)
		}
	}

	var waMessage *waE2E.Message
	var messageType string
	var payload string

	switch input.Type {
	case "text":
		if input.Text == "" {
			return model.Message{}, ErrInvalidPayload
		}
		waMessage = &waE2E.Message{
			Conversation: proto.String(input.Text),
		}
		messageType = "text"
		payload = input.Text

	case "image", "video":
		if len(input.MediaData) == 0 {
			return model.Message{}, ErrInvalidPayload
		}

		// Determinar tipo de mídia do WhatsMeow
		var mediaType whatsmeow.MediaType
		if input.Type == "image" {
			mediaType = whatsmeow.MediaImage
		} else {
			mediaType = whatsmeow.MediaVideo
		}

		// Upload da mídia
		uploadResp, err := client.Upload(ctx, input.MediaData, mediaType)
		if err != nil {
			return model.Message{}, fmt.Errorf("erro ao fazer upload da mídia: %w", err)
		}

		// Construir mensagem de mídia
		if input.Type == "image" {
			imageMsg := &waE2E.ImageMessage{
				URL:           &uploadResp.URL,
				DirectPath:    &uploadResp.DirectPath,
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    &uploadResp.FileLength,
				Mimetype:      proto.String(input.MediaType),
			}
			if input.Caption != "" {
				imageMsg.Caption = proto.String(input.Caption)
			}
			waMessage = &waE2E.Message{
				ImageMessage: imageMsg,
			}
		} else {
			videoMsg := &waE2E.VideoMessage{
				URL:           &uploadResp.URL,
				DirectPath:    &uploadResp.DirectPath,
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    &uploadResp.FileLength,
				Mimetype:      proto.String(input.MediaType),
			}
			if input.Caption != "" {
				videoMsg.Caption = proto.String(input.Caption)
			}
			waMessage = &waE2E.Message{
				VideoMessage: videoMsg,
			}
		}
		messageType = input.Type
		payload = fmt.Sprintf("media:%s", input.MediaType)

	case "audio":
		if len(input.MediaData) == 0 {
			return model.Message{}, ErrInvalidPayload
		}

		// Upload do áudio
		uploadResp, err := client.Upload(ctx, input.MediaData, whatsmeow.MediaAudio)
		if err != nil {
			return model.Message{}, fmt.Errorf("erro ao fazer upload do áudio: %w", err)
		}
		// Determinar se é PTT (voice message)
		// Alteração: Usar APENAS a flag explícita para dar controle total
		isPTT := input.PTT
		// || strings.HasPrefix(input.MediaType, "audio/ogg") || strings.Contains(input.MediaType, "opus")

		// || strings.HasPrefix(input.MediaType, "audio/ogg") || strings.Contains(input.MediaType, "opus")

		// Se for PTT, usar Waveform gerada, mas com visual agradável
		var waveform []byte
		var sidecar []byte

		if isPTT {
			// Gerar Waveform
			waveform = make([]byte, 64)
			for i := 0; i < 64; i++ {
				distFromCenter := math.Abs(float64(i) - 31.5)
				normalizedDist := distFromCenter / 32.0

				envelope := 1.0 - (normalizedDist * normalizedDist)
				if envelope < 0 {
					envelope = 0
				}

				baseHeight := envelope * 50.0
				noise := float64(rand.Intn(40))

				finalVal := baseHeight + noise
				if finalVal > 255 {
					finalVal = 255
				}

				waveform[i] = byte(finalVal)
			}

			sidecar = make([]byte, 16)
		}

		// Fix: WhatsApp exige mimetype explicito para OGG Opus como PTT
		finalMimeType := input.MediaType
		if isPTT && strings.Contains(input.MediaType, "audio/ogg") {
			finalMimeType = "audio/ogg; codecs=opus"
		}

		audioMsg := &waE2E.AudioMessage{
			URL:           &uploadResp.URL,
			DirectPath:    &uploadResp.DirectPath,
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    &uploadResp.FileLength,
			Mimetype:      proto.String(finalMimeType),
			ContextInfo: &waE2E.ContextInfo{
				Expiration: proto.Uint32(0),
			},
			PTT:               proto.Bool(isPTT),
			Seconds:           proto.Uint32(uint32(input.Seconds)),
			Waveform:          waveform,
			StreamingSidecar:  sidecar,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
		}
		waMessage = &waE2E.Message{
			AudioMessage: audioMsg,
		}
		messageType = "audio"
		payload = fmt.Sprintf("audio:%s", input.MediaType)

	case "document":
		if len(input.MediaData) == 0 {
			return model.Message{}, ErrInvalidPayload
		}

		// Upload do documento
		uploadResp, err := client.Upload(ctx, input.MediaData, whatsmeow.MediaDocument)
		if err != nil {
			return model.Message{}, fmt.Errorf("erro ao fazer upload do documento: %w", err)
		}

		// Determinar nome do arquivo
		fileName := input.FileName
		if fileName == "" {
			// Tentar extrair do mime type
			exts, _ := mime.ExtensionsByType(input.MediaType)
			if len(exts) > 0 {
				fileName = "document" + exts[0]
			} else {
				fileName = "document"
			}
		}

		docMsg := &waE2E.DocumentMessage{
			URL:           &uploadResp.URL,
			DirectPath:    &uploadResp.DirectPath,
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    &uploadResp.FileLength,
			Mimetype:      proto.String(input.MediaType),
			FileName:      proto.String(fileName),
		}
		if input.Caption != "" {
			docMsg.Caption = proto.String(input.Caption)
		}
		waMessage = &waE2E.Message{
			DocumentMessage: docMsg,
		}
		messageType = "document"
		payload = fmt.Sprintf("document:%s:%s", fileName, input.MediaType)

	default:
		return model.Message{}, fmt.Errorf("%w: %s", ErrUnsupportedMediaType, input.Type)
	}

	// Criar registro da mensagem
	message := model.Message{
		ID:         uuid.NewString(),
		InstanceID: input.InstanceID,
		To:         input.To,
		Type:       messageType,
		Payload:    payload,
		Status:     "sending",
	}

	// Salvar mensagem no banco ANTES de enviar (para ter histórico)
	msg, err := s.repo.Create(ctx, message)
	if err != nil {
		return model.Message{}, fmt.Errorf("erro ao salvar mensagem: %w", err)
	}

	// Enviar mensagem (síncrono - aguarda resposta do servidor)
	_, err = client.SendMessage(ctx, toJID, waMessage)
	if err != nil {
		// Atualizar status para failed
		msg.Status = "failed"
		// Tentar atualizar no banco (não crítico se falhar)
		_ = s.repo.Update(ctx, msg)
		return msg, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	// Sucesso - atualizar status
	msg.Status = "sent"
	// Opcional: salvar server_id e timestamp se quiser
	// resp.ID e resp.Timestamp estão disponíveis se necessário

	if err := s.repo.Update(ctx, msg); err != nil {
		// Log mas não retorna erro (mensagem já foi enviada)
	}

	return msg, nil
}

func (s *Service) List(ctx context.Context, instanceID string) ([]model.Message, error) {
	return s.repo.ListByInstance(ctx, instanceID)
}

// resolveJID tenta resolver o JID correto checando IsOnWhatsApp para números do Brasil
func (s *Service) resolveJID(ctx context.Context, client *whatsmeow.Client, phone string) (types.JID, error) {
	phone = strings.TrimSpace(phone)

	if phone == "" {
		return types.EmptyJID, errors.New("telefone vazio")
	}

	if strings.Contains(phone, "@g.us") || strings.Contains(phone, "@broadcast") {
		return types.ParseJID(phone)
	}

	if !strings.Contains(phone, "@") {
		phone = strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, phone)
	}

	if strings.Contains(phone, "@s.whatsapp.net") {
		phone = strings.TrimSuffix(phone, "@s.whatsapp.net")
	}

	if !strings.HasPrefix(phone, "55") {
		return types.ParseJID(phone + "@s.whatsapp.net")
	}

	candidates := []string{phone}

	if len(phone) == 13 {
		optionWithout9 := phone[:4] + phone[5:]
		candidates = append(candidates, optionWithout9)
	} else if len(phone) == 12 {
		optionWith9 := phone[:4] + "9" + phone[4:]
		candidates = append(candidates, optionWith9)
	}

	resp, err := client.IsOnWhatsApp(ctx, candidates)
	if err != nil {
		s.log.Warn("falha ao consultar IsOnWhatsApp, enviando original", zap.String("phone", phone), zap.Error(err))
		return types.ParseJID(phone + "@s.whatsapp.net")
	}

	for _, item := range resp {
		if item.JID.User != "" {
			return item.JID, nil
		}
	}

	return types.ParseJID(phone + "@s.whatsapp.net")
}
