package message

import (
	"sync"
	"time"
)

// Estado do que já foi sinalizado ao WhatsApp, para não repetir sinal que continua valendo.
//
// Antes toda mensagem reenviava `available` e reabria `composing`, mesmo quando os dois já
// estavam ativos do envio anterior. Além do custo de rede em cada envio, presença sempre-online é
// um dos padrões que a detecção de automação da Meta observa: quanto menos ruído repetido, melhor.
var (
	availableAt sync.Map // instanceID -> time.Time do último `available`
	composingAt sync.Map // instanceID + "|" + chatJID -> time.Time do último `composing`
)

const (
	// O `available` é estado do cliente e permanece até ser trocado ou a conexão cair. Como a
	// queda é tratada por evento (ver EsquecerPresencaDaInstancia), este TTL não é a garantia de
	// correção, é só uma rede contra reconexão que passe despercebida. Por isso pode ser folgado:
	// o whatsmeow força reconexão em até 3 minutos quando o keepalive falha, e essa reconexão
	// emite o evento que limpa este estado.
	availableTTL = 15 * time.Minute

	// No protocolo do WhatsApp Web, que é o que falamos aqui, a presença expira em torno de 10
	// segundos, e não nos 25 da API oficial (esse número vale para a Cloud API, que é outro
	// caminho). 8 s fica abaixo com margem para o relógio das duas pontas não coincidir; acima
	// disso o cache diria "ainda aberto" sobre um indicador que o servidor já dissolveu, e a
	// mensagem sairia sem sinal nenhum.
	//
	// Na prática este cache raramente entra em ação: todo envio termina em `paused`, que fecha o
	// indicador e limpa a marca. Ele vale para envios concorrentes no mesmo chat e para o caso de
	// o `paused` falhar.
	composingTTL = 8 * time.Second
)

func chaveDeChat(instanceID, chatJID string) string {
	return instanceID + "|" + chatJID
}

func expirou(m *sync.Map, chave string, ttl time.Duration) bool {
	valor, ok := m.Load(chave)
	if !ok {
		return true
	}
	quando, ok := valor.(time.Time)
	return !ok || time.Since(quando) >= ttl
}

// precisaAvailable diz se vale reenviar o `available` desta instância, marcando o envio.
func precisaAvailable(instanceID string) bool {
	if !expirou(&availableAt, instanceID, availableTTL) {
		return false
	}
	availableAt.Store(instanceID, time.Now())
	return true
}

// precisaComposing diz se vale reabrir o indicador de digitação neste chat, marcando a abertura.
func precisaComposing(instanceID, chatJID string) bool {
	chave := chaveDeChat(instanceID, chatJID)
	if !expirou(&composingAt, chave, composingTTL) {
		return false
	}
	composingAt.Store(chave, time.Now())
	return true
}

// esqueceComposing é chamado ao mandar `paused`: o indicador não está mais aberto, então o próximo
// envio precisa reabri-lo de verdade.
func esqueceComposing(instanceID, chatJID string) {
	composingAt.Delete(chaveDeChat(instanceID, chatJID))
}

// EsquecerPresencaDaInstancia limpa o estado ao desconectar: depois de reconectar, nada do que foi
// sinalizado antes continua valendo do outro lado.
func EsquecerPresencaDaInstancia(instanceID string) {
	availableAt.Delete(instanceID)
	prefixo := instanceID + "|"
	composingAt.Range(func(k, _ any) bool {
		if chave, ok := k.(string); ok && len(chave) > len(prefixo) && chave[:len(prefixo)] == prefixo {
			composingAt.Delete(chave)
		}
		return true
	})
}
