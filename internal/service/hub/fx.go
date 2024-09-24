package hub

import (
	"github.com/brucexc/pray-to-earn/internal/provider"
	"go.uber.org/fx"
)

var Module = fx.Options(
	//fx.Provide(provider.ProvideDatabaseClient),
	fx.Provide(provider.ProvideEthereumClient),
	fx.Provide(provider.ProvideRedisClient),
)
