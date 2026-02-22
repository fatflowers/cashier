package main

// @title           Cashier Backend API
// @version         1.0
// @description     Payment processing backend API with health monitoring.
// @termsOfService  http://example.com/terms/

// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8888
// @BasePath  /

import (
	"context"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/fatflowers/cashier/internal/app"
)

func main() {
	// Allow graceful stop with SIGINT/SIGTERM handled by fx
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	a := fx.New(app.Module)
	startCtx, cancel := context.WithTimeout(context.Background(), app.DefaultStartTimeout)
	defer cancel()
	if err := a.Start(startCtx); err != nil {
		// Logging might not be ready; fallback to zap example
		zap.NewExample().Sugar().Errorf("failed to start app: %v", err)
		exitCode = 1
		return
	}

	// Block until signal
	<-a.Done()

	stopCtx, cancel2 := context.WithTimeout(context.Background(), app.DefaultStopTimeout)
	defer cancel2()
	if err := a.Stop(stopCtx); err != nil {
		zap.NewExample().Sugar().Errorf("failed to stop app: %v", err)
		exitCode = 1
		return
	}
}
