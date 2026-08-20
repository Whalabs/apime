package message

import (
	"context"
	"testing"
	"time"
)

/*
Regras do "digitando" simulado.

O indicador existe para a conversa não parecer automática, e é por isso que ele nunca some por
completo. O que estas regras evitam é o oposto: fazer o contato esperar duas vezes pela mesma
espera, ou travar um áudio de 50 s por 30 s antes de enviar. Erro aqui é silencioso, aparece como
"o sistema está lento" e ninguém liga ao cálculo.
*/

func TestDescontarEsperaMantemPiso(t *testing.T) {
	// O contato já esperou muito mais do que a digitação levaria: ainda assim o indicador aparece.
	if got := descontarEspera(8*time.Second, 40*time.Second); got != pisoDeDigitacao {
		t.Fatalf("esperava o piso %v, veio %v", pisoDeDigitacao, got)
	}
}

func TestDescontarEsperaSubtraiOQueJaPassou(t *testing.T) {
	if got := descontarEspera(8*time.Second, 3*time.Second); got != 5*time.Second {
		t.Fatalf("esperava 5s, veio %v", got)
	}
}

func TestDescontarEsperaSemEsperaAnteriorNaoMuda(t *testing.T) {
	if got := descontarEspera(6*time.Second, 0); got != 6*time.Second {
		t.Fatalf("esperava 6s, veio %v", got)
	}
}

func TestConversaAbertaNaoGanhaAcrescimo(t *testing.T) {
	// Caso comum da operação: o contato escreveu primeiro, e responder rápido é o que um humano
	// faria. É o único caso em que a espera diminui.
	if got := ajustarPorFamiliaridade(5*time.Second, true); got != 5*time.Second {
		t.Fatalf("conversa aberta não deveria ganhar acréscimo, veio %v", got)
	}
}

func TestPrimeiroContatoMantemOAcrescimoDeAntes(t *testing.T) {
	// Primeiro contato sem inbound é o perfil de campanha, e o delay do envio SOMA ao intervalo
	// que a campanha já espera entre destinatários. Encurtá-lo aceleraria toda campanha já
	// configurada sem ninguém pedir, que é como se aumenta taxa de disparo em silêncio.
	base := 5 * time.Second
	for i := 0; i < 50; i++ {
		got := ajustarPorFamiliaridade(base, false)
		if got < base+1500*time.Millisecond || got > base+2500*time.Millisecond {
			t.Fatalf("acréscimo fora da faixa de 1,5 a 2,5 s: %v", got-base)
		}
	}
}

func TestPrimeiroContatoNaoPassaDoQueEraAntes(t *testing.T) {
	// Fora da janela de reconexão, o pior caso tem de continuar sendo o de antes: teto do texto
	// (8 s) mais o acréscimo fixo.
	if got := ajustarPorFamiliaridade(8*time.Second, false); got > 10500*time.Millisecond {
		t.Fatalf("pior caso acima do que já existia: %v", got)
	}
}

func TestFatorDeReconexaoCaiComOTempo(t *testing.T) {
	janela := 60 * time.Second
	inicio := fatorDeReconexao(0, janela)
	meio := fatorDeReconexao(30*time.Second, janela)
	fim := fatorDeReconexao(60*time.Second, janela)

	if inicio <= meio || meio <= fim {
		t.Fatalf("o freio deveria diminuir: %v, %v, %v", inicio, meio, fim)
	}
	if fim != 1.0 {
		t.Fatalf("passada a janela não há freio, esperava 1.0, veio %v", fim)
	}
	// Depois da janela nada muda, por mais tempo que passe.
	if got := fatorDeReconexao(10*time.Minute, janela); got != 1.0 {
		t.Fatalf("esperava 1.0 muito depois da janela, veio %v", got)
	}
}

func TestDelayDeAudioRespeitaOTeto(t *testing.T) {
	// Áudio de 50 s: a conta pela duração inteira era o que produzia os picos de quase um minuto.
	got := calculatePresenceDelay(SendInput{Type: "audio", Seconds: 50})
	if got > tetoDeGravacao+tetoDeGravacao/5 { // +20% do jitter
		t.Fatalf("áudio longo deveria respeitar o teto de %v, veio %v", tetoDeGravacao, got)
	}
}

func TestDelayDeAudioCurtoUsaADuracao(t *testing.T) {
	got := calculatePresenceDelay(SendInput{Type: "audio", Seconds: 4})
	if got < 3*time.Second || got > 5*time.Second {
		t.Fatalf("áudio de 4s deveria gerar algo perto disso, veio %v", got)
	}
}

func TestCacheDePresencaRespeitaJanela(t *testing.T) {
	availableAt.Delete("inst-1")
	if !precisaAvailable("inst-1") {
		t.Fatal("primeira vez precisa sinalizar")
	}
	if precisaAvailable("inst-1") {
		t.Fatal("dentro da janela não deveria repetir")
	}

	composingAt.Delete(chaveDeChat("inst-1", "chat-a"))
	if !precisaComposing("inst-1", "chat-a") {
		t.Fatal("primeira vez precisa abrir o indicador")
	}
	if precisaComposing("inst-1", "chat-a") {
		t.Fatal("indicador já aberto não deveria ser reaberto")
	}
	// Outro chat da mesma instância é independente: o indicador é por conversa.
	if !precisaComposing("inst-1", "chat-b") {
		t.Fatal("chat diferente deveria abrir o próprio indicador")
	}

	// Fechar com `paused` faz o próximo envio reabrir de verdade.
	esqueceComposing("inst-1", "chat-a")
	if !precisaComposing("inst-1", "chat-a") {
		t.Fatal("depois de paused o indicador precisa ser reaberto")
	}
}

func TestEsquecerPresencaDaInstancia(t *testing.T) {
	precisaAvailable("inst-2")
	precisaComposing("inst-2", "chat-x")
	precisaComposing("inst-3", "chat-x")

	EsquecerPresencaDaInstancia("inst-2")

	if !precisaAvailable("inst-2") {
		t.Fatal("após desconectar, o available precisa ser reenviado")
	}
	if !precisaComposing("inst-2", "chat-x") {
		t.Fatal("após desconectar, o indicador precisa ser reaberto")
	}
	// Instância vizinha não pode ser afetada pela limpeza da outra.
	if precisaComposing("inst-3", "chat-x") {
		t.Fatal("limpeza vazou para outra instância")
	}
}

func TestDormirDesisteQuandoCancelado(t *testing.T) {
	// A simulação corre em paralelo com o preparo do envio: abortado o envio, o indicador não pode
	// continuar aparecendo para o contato.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	inicio := time.Now()
	if dormir(ctx, 5*time.Second) {
		t.Fatal("contexto cancelado deveria interromper a espera")
	}
	if time.Since(inicio) > time.Second {
		t.Fatalf("deveria retornar de imediato, levou %v", time.Since(inicio))
	}
}

func TestDormirCompletaQuandoNaoCancelado(t *testing.T) {
	if !dormir(context.Background(), 10*time.Millisecond) {
		t.Fatal("sem cancelamento a espera deveria completar")
	}
}

// O jitter gaussiano tem cauda infinita: sem corte, uma amostra rara viraria uma espera absurda
// (ou negativa). O corte em dois desvios é o que torna a duração previsível no pior caso.
func TestJitterGaussianoFicaDentroDoCorte(t *testing.T) {
	for i := 0; i < 10000; i++ {
		f := jitterGaussiano(0.20)
		if f < 0.6-1e-9 || f > 1.4+1e-9 {
			t.Fatalf("fator fora do corte de dois desvios: %v", f)
		}
	}
}

// O ponto do gaussiano é agrupar perto da média, não só respeitar os limites: um uniforme também
// caberia no corte. A maioria das amostras tem que cair dentro de um desvio.
func TestJitterGaussianoAgrupaNaMedia(t *testing.T) {
	dentro := 0
	const amostras = 10000
	for i := 0; i < amostras; i++ {
		if f := jitterGaussiano(0.10); f > 0.9 && f < 1.1 {
			dentro++
		}
	}
	if dentro < amostras*6/10 {
		t.Fatalf("distribuição não agrupou na média: só %d de %d dentro de um desvio", dentro, amostras)
	}
}
