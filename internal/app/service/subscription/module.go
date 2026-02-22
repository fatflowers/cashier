package subscription

import "go.uber.org/fx"

// Module exposes the subscription service via Fx.
var Module = fx.Options(
	fx.Provide(NewService),
)
