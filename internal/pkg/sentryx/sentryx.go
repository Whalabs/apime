// Package sentryx é um wrapper fino sobre github.com/getsentry/sentry-go que
// concentra inicialização, no-op silencioso quando SENTRY_DSN não está setado,
// captura de erros do whatsmeow (via zap hook) e middleware do Gin.
//
// Sem DSN configurado, todas as funções viram no-op — não há overhead e nenhuma
// chamada de rede é feita.
package sentryx

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
)

// Config é o subset que consumimos do internal/config.SentryConfig — duplicado
// como struct simples para não criar import cycle.
type Config struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64
	TracesSampleRate float64
	ServerName       string
}

var enabled bool

// IsEnabled reporta se o Sentry foi inicializado com DSN válido.
func IsEnabled() bool { return enabled }

// Init inicializa o Sentry quando há DSN configurado. Sem DSN é no-op
// silencioso. Erros de init são apenas reportados pelo caller (log) — não
// derrubam o processo.
func Init(cfg Config) error {
	if cfg.DSN == "" {
		enabled = false
		return nil
	}
	opts := sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		ServerName:       cfg.ServerName,
		SampleRate:       cfg.SampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		AttachStacktrace: true,
		EnableTracing:    cfg.TracesSampleRate > 0,
	}
	if opts.SampleRate == 0 {
		opts.SampleRate = 1.0
	}
	if err := sentry.Init(opts); err != nil {
		enabled = false
		return err
	}
	enabled = true
	return nil
}

// Flush bloqueia até pendências serem enviadas ou o timeout estourar.
func Flush(timeout time.Duration) {
	if !enabled {
		return
	}
	sentry.Flush(timeout)
}

// CaptureError envia um erro com tags opcionais.
func CaptureError(err error, tags map[string]string) {
	if !enabled || err == nil {
		return
	}
	hub := sentry.CurrentHub().Clone()
	hub.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		hub.CaptureException(err)
	})
}

// CaptureMessage envia uma mensagem com level.
func CaptureMessage(msg string, level sentry.Level, tags map[string]string) {
	if !enabled || msg == "" {
		return
	}
	hub := sentry.CurrentHub().Clone()
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		hub.CaptureMessage(msg)
	})
}

// Recover deve ser chamado via defer em goroutines críticas para reportar
// panics ao Sentry e re-lançar para preservar o comportamento original.
func Recover() {
	if r := recover(); r != nil {
		if enabled {
			sentry.CurrentHub().Recover(r)
			sentry.Flush(2 * time.Second)
		}
		panic(r)
	}
}

// RecoverWithContext igual a Recover, com escopo derivado de ctx.
func RecoverWithContext(ctx context.Context) {
	if r := recover(); r != nil {
		if enabled {
			sentry.GetHubFromContext(ctx).Recover(r)
			sentry.Flush(2 * time.Second)
		}
		panic(r)
	}
}
