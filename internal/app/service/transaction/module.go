package transaction

import "go.uber.org/fx"

// Module exposes the transaction service via Fx.
var Module = fx.Options(
	fx.Provide(NewAppleTransactionManager),
	fx.Provide(NewService),
)
