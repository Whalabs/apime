-- Migração para rastreamento de entrega de mensagens
ALTER TABLE message_queue ADD COLUMN whatsapp_id TEXT;
ALTER TABLE message_queue ADD COLUMN delivered_at TIMESTAMPTZ;

-- Índice para busca rápida por ID do WhatsApp (usado nos receipts)
CREATE INDEX IF NOT EXISTS idx_message_queue_whatsapp_id ON message_queue(whatsapp_id);
