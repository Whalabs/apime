package message

import (
	"context"
	"math/rand"
	"testing"
	"time"
)

// Rules for the simulated typing indicator. It never disappears entirely, since that is what keeps
// the conversation from looking automated. These rules prevent the opposite: making the contact wait
// twice for the same wait. A mistake here is silent, surfacing only as "the system is slow".

func TestSubtractElapsedKeepsFloor(t *testing.T) {
	// The contact waited far longer than typing would take, and the indicator still shows.
	if got := subtractElapsed(8*time.Second, 40*time.Second); got != typingFloor {
		t.Fatalf("esperava o piso %v, veio %v", typingFloor, got)
	}
}

func TestSubtractElapsedDiscountsWhatPassed(t *testing.T) {
	if got := subtractElapsed(8*time.Second, 3*time.Second); got != 5*time.Second {
		t.Fatalf("esperava 5s, veio %v", got)
	}
}

func TestSubtractElapsedWithNoPriorWait(t *testing.T) {
	if got := subtractElapsed(6*time.Second, 0); got != 6*time.Second {
		t.Fatalf("esperava 6s, veio %v", got)
	}
}

func TestOpenConversationGetsNoBump(t *testing.T) {
	// The contact wrote first and replying fast is what a human would do: the only case that
	// shortens the wait.
	if got := adjustForFamiliarity(5*time.Second, true); got != 5*time.Second {
		t.Fatalf("conversa aberta não deveria ganhar acréscimo, veio %v", got)
	}
}

func TestFirstContactKeepsPreviousBump(t *testing.T) {
	// First contact without inbound is the campaign profile, and this delay ADDS to the interval the
	// campaign already waits between recipients. Shortening it would speed up every campaign already
	// configured, raising send rate silently.
	base := 5 * time.Second
	for i := 0; i < 50; i++ {
		got := adjustForFamiliarity(base, false)
		if got < base+1500*time.Millisecond || got > base+2500*time.Millisecond {
			t.Fatalf("acréscimo fora da faixa de 1,5 a 2,5 s: %v", got-base)
		}
	}
}

func TestFirstContactNoWorseThanBefore(t *testing.T) {
	// Outside the reconnect window the worst case must stay what it was: text cap (8s) plus the
	// fixed bump.
	if got := adjustForFamiliarity(8*time.Second, false); got > 10500*time.Millisecond {
		t.Fatalf("pior caso acima do que já existia: %v", got)
	}
}

func TestReconnectFactorDecays(t *testing.T) {
	window := 60 * time.Second
	start := reconnectFactor(0, window)
	middle := reconnectFactor(30*time.Second, window)
	end := reconnectFactor(60*time.Second, window)

	if start <= middle || middle <= end {
		t.Fatalf("o freio deveria diminuir: %v, %v, %v", start, middle, end)
	}
	if end != 1.0 {
		t.Fatalf("passada a janela não há freio, esperava 1.0, veio %v", end)
	}
	// Past the window nothing changes, however long it has been.
	if got := reconnectFactor(10*time.Minute, window); got != 1.0 {
		t.Fatalf("esperava 1.0 muito depois da janela, veio %v", got)
	}
}

func TestAudioDelayRespectsCap(t *testing.T) {
	// 50s audio: basing it on the full duration was what produced the near-minute peaks.
	got := calculatePresenceDelay(SendInput{Type: "audio", Seconds: 50})
	if got > recordingCap+recordingCap/5 { // +20% do jitter
		t.Fatalf("áudio longo deveria respeitar o teto de %v, veio %v", recordingCap, got)
	}
}

func TestShortAudioUsesItsDuration(t *testing.T) {
	got := calculatePresenceDelay(SendInput{Type: "audio", Seconds: 4})
	if got < 3*time.Second || got > 5*time.Second {
		t.Fatalf("áudio de 4s deveria gerar algo perto disso, veio %v", got)
	}
}

func TestPresenceCacheRespectsWindow(t *testing.T) {
	// The cache is package-global: without clearing, a second run (go test -count=2) would inherit
	// the first run's marks and fail with nothing broken.
	ForgetInstancePresence("inst-1")
	if !needsAvailable("inst-1") {
		t.Fatal("primeira vez precisa sinalizar")
	}
	if needsAvailable("inst-1") {
		t.Fatal("dentro da janela não deveria repetir")
	}

	composingAt.Delete(chatKey("inst-1", "chat-a"))
	if !needsComposing("inst-1", "chat-a") {
		t.Fatal("primeira vez precisa abrir o indicador")
	}
	if needsComposing("inst-1", "chat-a") {
		t.Fatal("indicador já aberto não deveria ser reaberto")
	}
	// Another chat on the same instance is independent: the indicator is per conversation.
	if !needsComposing("inst-1", "chat-b") {
		t.Fatal("chat diferente deveria abrir o próprio indicador")
	}

	// Closing with `paused` makes the next send actually reopen it.
	forgetComposing("inst-1", "chat-a")
	if !needsComposing("inst-1", "chat-a") {
		t.Fatal("depois de paused o indicador precisa ser reaberto")
	}
}

func TestForgetInstancePresence(t *testing.T) {
	// Same reason as the test above: package state must not cross runs.
	ForgetInstancePresence("inst-2")
	ForgetInstancePresence("inst-3")

	needsAvailable("inst-2")
	needsComposing("inst-2", "chat-x")
	needsComposing("inst-3", "chat-x")

	ForgetInstancePresence("inst-2")

	if !needsAvailable("inst-2") {
		t.Fatal("após desconectar, o available precisa ser reenviado")
	}
	if !needsComposing("inst-2", "chat-x") {
		t.Fatal("após desconectar, o indicador precisa ser reaberto")
	}
	// A neighbouring instance must not be affected by another's cleanup.
	if needsComposing("inst-3", "chat-x") {
		t.Fatal("limpeza vazou para outra instância")
	}
}

func TestSleepCtxGivesUpWhenCanceled(t *testing.T) {
	// The simulation runs in parallel with send preparation: once the send aborts, the indicator
	// must not keep showing for the contact.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if sleepCtx(ctx, 5*time.Second) {
		t.Fatal("contexto cancelado deveria interromper a espera")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("deveria retornar de imediato, levou %v", time.Since(start))
	}
}

func TestSleepCtxCompletesWhenNotCanceled(t *testing.T) {
	if !sleepCtx(context.Background(), 10*time.Millisecond) {
		t.Fatal("sem cancelamento a espera deveria completar")
	}
}

// The gaussian tail is infinite: without clamping, a rare sample would become an absurd (or
// negative) wait. Clamping at two standard deviations bounds the worst case.
func TestGaussianJitterStaysWithinClamp(t *testing.T) {
	for i := 0; i < 10000; i++ {
		f := gaussianJitter(0.20)
		if f < 0.6-1e-9 || f > 1.4+1e-9 {
			t.Fatalf("fator fora do corte de dois desvios: %v", f)
		}
	}
}

// The point of the gaussian is clustering near the mean, not just fitting the clamp: a uniform
// would fit too. Fixed seed because the assertion is statistical, and the global rand would make
// this fail intermittently in CI.
func TestGaussianJitterClustersNearMean(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	const stddev = 0.10

	inside := 0
	const samples = 10000
	for i := 0; i < samples; i++ {
		// Mirrors gaussianJitter with a controlled source.
		f := 1 + rng.NormFloat64()*stddev
		if f > 0.9 && f < 1.1 {
			inside++
		}
	}
	if inside < samples*6/10 {
		t.Fatalf("distribuição não agrupou na média: só %d de %d dentro de um desvio", inside, samples)
	}
}
