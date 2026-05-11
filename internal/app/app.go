package app

import (
	"context"
	"os"

	"box-link/internal/config"
	"box-link/internal/iface"
	"box-link/internal/logx"
	"box-link/internal/netops"
	"box-link/internal/syscmd"

	"github.com/sirupsen/logrus"
)

type App struct {
	Config   config.Config
	Log      *logrus.Logger
	LogStore *logx.Store
	Runner   syscmd.Runner
}

func New(cfg config.Config) *App {
	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	logger.SetLevel(parseLevel(cfg.LogLevel))
	logger.SetFormatter(&logx.Formatter{
		ColorEnabled: logx.IsTerminal(os.Stderr) && os.Getenv("NO_COLOR") == "",
	})

	store := logx.NewStore()
	logger.AddHook(store)

	logger.Infof("using config file: %s", cfg.ConfigPath)

	return &App{
		Config:   cfg,
		Log:      logger,
		LogStore: store,
		Runner:   syscmd.New(),
	}
}

func (a *App) ResolveIface(ctx context.Context) (iface.Result, error) {
	return iface.Resolve(ctx, a.Config, a.Runner, a.Log)
}

func (a *App) On(ctx context.Context) (netops.OnResult, error) {
	return netops.On(ctx, a.Config, a.Runner, a.Log)
}

func (a *App) Off(ctx context.Context) (netops.OffResult, error) {
	return netops.Off(ctx, a.Config, a.Runner, a.Log)
}

func (a *App) Status(ctx context.Context) (netops.StatusResult, error) {
	return netops.Status(ctx, a.Config, a.Runner, a.Log)
}

func (a *App) Doctor(ctx context.Context) (netops.DoctorResult, error) {
	return netops.Doctor(ctx, a.Config, a.Runner, a.Log)
}

func parseLevel(s string) logrus.Level {
	level, err := logrus.ParseLevel(s)
	if err != nil {
		return logrus.InfoLevel
	}
	return level
}
