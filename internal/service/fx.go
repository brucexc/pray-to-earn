package service

import (
	"context"
	"github.com/brucexc/pray-to-earn/internal/constant"
	"github.com/brucexc/pray-to-earn/internal/provider"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func NewServer(options ...fx.Option) *fx.App {
	return fx.New(
		fx.Options(options...),
		fx.Provide(provider.ProvideConfig),
		fx.Invoke(InjectLifecycle),
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ZapLogger{
				Logger: zap.L(),
			}
		}),
	)
}

func InjectLifecycle(lifecycle fx.Lifecycle, server Server) {
	constant.ServiceName = server.Name()

	hook := fx.Hook{
		OnStart: func(ctx context.Context) error {
			return server.Run(ctx)
		},
	}

	lifecycle.Append(hook)
}
