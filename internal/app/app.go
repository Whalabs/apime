package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/open-apime/apime/internal/config"
)

type App struct {
	cfg    config.Config
	logger *zap.Logger
	server *http.Server
}

func New(cfg config.Config, logger *zap.Logger, handler http.Handler) *App {
	return &App{
		cfg:    cfg,
		logger: logger,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.App.Port),
			Handler: handler,
		},
	}
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("iniciando servidor HTTP",
		zap.String("port", a.cfg.App.Port),
		zap.String("env", a.cfg.App.Env),
		zap.String("addr", a.server.Addr),
	)
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.logger.Error("erro ao iniciar servidor",
			zap.String("port", a.cfg.App.Port),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("encerrando servidor HTTP",
		zap.String("port", a.cfg.App.Port),
	)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("erro ao encerrar servidor",
			zap.Error(err),
		)
		return err
	}
	a.logger.Info("servidor encerrado com sucesso")
	return nil
}
