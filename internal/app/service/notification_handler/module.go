package notification_handler

import "go.uber.org/fx"

var Module = fx.Options(
    fx.Provide(NewNotificationHandler),
)

