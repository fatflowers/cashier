package notification_log

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(New),
)
