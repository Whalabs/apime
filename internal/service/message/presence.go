package message

import (
	"context"
	"math/rand"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

/*
jitterGaussiano devolve um fator em torno de 1,0 para variar uma duração.

O jitter uniforme espalha os valores igualmente por toda a faixa, e isso é reconhecível: gente não
distribui o próprio tempo assim. A normal agrupa perto da média, com desvios grandes raros, que é o
formato de tempo humano. As pontas são cortadas em dois desvios, porque a cauda da normal chega a
valores absurdos e aqui isso viraria uma espera esquisita.
*/
func jitterGaussiano(desvio float64) float64 {
	fator := 1 + rand.NormFloat64()*desvio
	if minimo := 1 - 2*desvio; fator < minimo {
		fator = minimo
	}
	if maximo := 1 + 2*desvio; fator > maximo {
		fator = maximo
	}
	return fator
}

/*
dormir espera, mas desiste se a requisição for cancelada.

A simulação passou a rodar em paralelo com o preparo da mensagem, então ela pode continuar de pé
depois de o envio ter sido abortado (mídia inválida, tipo não suportado, cliente desistiu). Com
time.Sleep puro, o indicador de digitação seguiria aparecendo para o contato por vários segundos
depois de nada ter sido enviado. Devolve false quando foi cancelada.
*/
func dormir(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// simulatePresenceDelay handles the typing/recording delay with optional micro-pauses.
// For text: breaks "typing..." into segments with brief pauses (type → pause → type → send).
// For audio: continuous recording without pauses (humans hold the record button).
// The initial ChatPresenceComposing must already be sent before calling this.
func simulatePresenceDelay(ctx context.Context, client *whatsmeow.Client, toJID types.JID, media types.ChatPresenceMedia, totalDelay time.Duration) {
	// Audio: continuous recording, no micro-pauses
	if media == types.ChatPresenceMediaAudio || totalDelay < 3*time.Second {
		dormir(ctx, totalDelay)
		return
	}

	// Text: micro-pauses for delays > 3s
	numPauses := 1
	if totalDelay > 5*time.Second {
		numPauses = 1 + rand.Intn(2) // 1-2
	}
	if totalDelay > 7*time.Second {
		numPauses = 2 + rand.Intn(2) // 2-3
	}

	// Pausas em torno de 500 ms, agrupadas na média em vez de espalhadas de 300 a 700.
	pauseDurations := make([]time.Duration, numPauses)
	var totalPauseTime time.Duration
	for i := range pauseDurations {
		p := time.Duration(500 * jitterGaussiano(0.20) * float64(time.Millisecond))
		pauseDurations[i] = p
		totalPauseTime += p
	}

	// Distribute remaining time across typing segments
	typingTime := totalDelay - totalPauseTime
	numSegments := numPauses + 1
	avgSegment := typingTime / time.Duration(numSegments)

	for i := 0; i < numSegments; i++ {
		// Cada trecho varia em torno da média do segmento, não uniformemente dentro dela.
		segDuration := time.Duration(float64(avgSegment) * jitterGaussiano(0.20))
		if segDuration < 400*time.Millisecond {
			segDuration = 400 * time.Millisecond
		}

		if !dormir(ctx, segDuration) {
			return
		}

		if i < numPauses {
			_ = client.SendChatPresence(ctx, toJID, types.ChatPresencePaused, media)
			if !dormir(ctx, pauseDurations[i]) {
				return
			}
			_ = client.SendChatPresence(ctx, toJID, types.ChatPresenceComposing, media)
		}
	}
}

func presenceMediaType(msgType string) types.ChatPresenceMedia {
	if msgType == "audio" {
		return types.ChatPresenceMediaAudio
	}
	return types.ChatPresenceMediaText
}

// Piso do indicador: mesmo quando o contato já esperou bastante, a digitação é sinalizada por um
// instante. Ela é um dos sinais que distinguem conversa humana de automação, e sair enviando sem
// nenhum indicador é justamente o padrão que a detecção da Meta observa.
const pisoDeDigitacao = 1200 * time.Millisecond

// Teto do "gravando". A conta pela duração inteira do áudio produzia esperas de 30 s (a origem dos
// picos de quase um minuto no envio). Ninguém cronometra se um áudio de 50 s levou 50 ou 15
// segundos sendo gravado; o que se percebe é a demora até a mensagem chegar.
const tetoDeGravacao = 15 * time.Second

/*
Ajuste por familiaridade com o contato.

Conversa em andamento é onde a demora atrapalha a operação e onde o risco é menor: o contato
escreveu primeiro, e responder rápido é o que um humano faria. Só ali o delay é reduzido.

Fora dela, o comportamento é EXATAMENTE o de antes: o acréscimo fixo de 1,5 a 2,5 s da "conversa
nova". Isso é deliberado. Primeiro contato sem inbound é o perfil de campanha, e o delay do envio
soma ao intervalo que a campanha já espera entre destinatários: encurtá-lo aceleraria toda campanha
já configurada, sem ninguém ter pedido. Quem calibrou aqueles intervalos calibrou com esta espera
embutida, e mudar por baixo é aumentar a taxa de disparo em silêncio, que é o que gera bloqueio.

`temSessao` deixa de alterar o tempo e continua só descrevendo o caso nos logs: a distinção que
importa para o risco é ter ou não conversa aberta.
*/
func ajustarPorFamiliaridade(base time.Duration, temInboundRecente bool) time.Duration {
	if temInboundRecente {
		return base
	}
	return base + time.Duration(1500+rand.Intn(1000))*time.Millisecond
}

/*
Desconta o tempo que o contato JÁ esperou desde a última mensagem dele.

Quem demora a responder, responde: somar o delay proporcional por cima de uma espera que já
aconteceu não humaniza, atrasa. O piso garante que o indicador continue aparecendo.
*/
func descontarEspera(delay, decorrido time.Duration) time.Duration {
	restante := delay - decorrido
	if restante < pisoDeDigitacao {
		return pisoDeDigitacao
	}
	return restante
}

/*
Freio dos primeiros segundos após conectar.

Voltar de uma reconexão despejando mensagens no ritmo normal é padrão de automação. O fator cai de
2x para 1x ao longo da janela de estabilização, em vez de mudar de degrau.
*/
func fatorDeReconexao(desdeConexao time.Duration, janela time.Duration) float64 {
	if desdeConexao >= janela || janela <= 0 {
		return 1.0
	}
	proporcaoRestante := 1.0 - float64(desdeConexao)/float64(janela)
	return 1.0 + proporcaoRestante
}

func calculatePresenceDelay(input SendInput) time.Duration {
	var base int
	switch input.Type {
	case "text":
		// ~40ms per char, min 1.5s, max 8s
		base = len(input.Text) * 40
		if base < 1500 {
			base = 1500
		}
		if base > 8000 {
			base = 8000
		}
	case "audio":
		// Based on audio duration (seconds), capped — ver tetoDeGravacao.
		if input.Seconds > 0 {
			base = input.Seconds * 1000
			if base > int(tetoDeGravacao/time.Millisecond) {
				base = int(tetoDeGravacao / time.Millisecond)
			}
		} else {
			base = 3000
		}
	case "image", "video":
		// Base delay + size factor + caption
		sizeMB := len(input.MediaData) / (1024 * 1024)
		base = 2000 + sizeMB*300
		if base > 8000 {
			base = 8000
		}
		if input.Caption != "" {
			extra := len(input.Caption) * 40
			if extra > 5000 {
				extra = 5000
			}
			base += extra
		}
	case "document":
		base = 1500
		if input.Caption != "" {
			extra := len(input.Caption) * 40
			if extra > 5000 {
				extra = 5000
			}
			base += extra
		}
	default:
		base = 1500
	}
	// Jitter em torno de ±10%, agrupado na média (ver jitterGaussiano).
	return time.Duration(float64(base)*jitterGaussiano(0.10)) * time.Millisecond
}
