package server

import (
	"context"
	"fmt"
	"github.com/fatflowers/cashier/docs"
	"github.com/fatflowers/cashier/internal/app/api/handlers"
	nh "github.com/fatflowers/cashier/internal/app/service/notification_handler"
	"github.com/fatflowers/cashier/internal/app/service/statistics"
	subsvc "github.com/fatflowers/cashier/internal/app/service/subscription"
	"github.com/fatflowers/cashier/internal/app/service/transaction"
	cfgpkg "github.com/fatflowers/cashier/pkg/config"
	"net/http"
	"time"

	mw "github.com/fatflowers/cashier/internal/app/api/middleware"

	metrics "github.com/fatflowers/cashier/pkg/metrics"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func newEngine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	// Add request tracing middleware only; request logger & access log are attached per group in registerRoutes
	r.Use(mw.TraceMiddleware())
	return r
}

func registerRoutes(r *gin.Engine, log *zap.SugaredLogger, notifHandler *nh.NotificationHandler, txMgr transaction.TransactionManager, sub *subsvc.Service, cfg *cfgpkg.Config, stats *statistics.Service) {
	// Prometheus metrics
	if cfg != nil && cfg.MetricsAddr != "" {
		p := metrics.NewPrometheus(metrics.NewPrometheusOptions{
			ReqCntURLLabelMappingFn: func(c *gin.Context) string {
				if fp := c.FullPath(); fp != "" {
					return fp
				}
				return c.Request.URL.Path
			},
			Logger: log,
		})
		p.SetListenAddress(cfg.MetricsAddr)
		p.Use(r)

		log.Infow("metrics started", "addr", cfg.MetricsAddr)
	}
	// Public group: request logger + access log
	pub := r.Group("/")
	pub.Use(mw.RequestLoggerMiddleware(log), mw.AccessLogMiddleware())
	handlers.RegisterHealthRoutes(pub)
	// Swagger UI
	docs.SwaggerInfo.BasePath = "/"
	pub.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Protected group using auth middleware
	apiV1 := r.Group("/api/v1")
	apiV1.Use(mw.RequestLoggerMiddleware(log), mw.AccessLogMiddleware())

	// Admin payment APIs
	handlers.RegisterAdminPaymentRoutes(apiV1.Group("/admin"), txMgr, cfg, stats, sub)

	// Payment v2 APIs
	apiV2Payment := r.Group("/api/v2/payment")
	apiV2Payment.Use(mw.RequestLoggerMiddleware(log), mw.AccessLogMiddleware())
	handlers.RegisterPaymentV2Routes(apiV2Payment, txMgr, notifHandler)
}

func runServer(lc fx.Lifecycle, log *zap.SugaredLogger, cfg *cfgpkg.Config, r *gin.Engine) {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 5 * time.Second}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Infow("starting HTTP server", "addr", addr)
			go func() {
				if err := srv.ListenAndServe(); err != nil {
					log.Errorf("server error: %v", err)
					panic(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Infow("stopping HTTP server")
			shutdownCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		},
	})
}

var Module = fx.Options(
	fx.Provide(newEngine),
	fx.Invoke(registerRoutes),
	fx.Invoke(runServer),
)
