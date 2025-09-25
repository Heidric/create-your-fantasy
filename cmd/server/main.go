package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Heidric/create-your-fantasy/internal/config"
	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/logger"
	"github.com/Heidric/create-your-fantasy/internal/server"
	"github.com/Heidric/create-your-fantasy/internal/services/auth"
	"github.com/Heidric/create-your-fantasy/internal/services/moderation"
	"github.com/Heidric/create-your-fantasy/internal/services/profile"
	"github.com/Heidric/create-your-fantasy/internal/services/report"
	"github.com/Heidric/create-your-fantasy/internal/services/session"
	"github.com/Heidric/create-your-fantasy/internal/storage/postgres"
	"github.com/Heidric/create-your-fantasy/pkg/pgx"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	runner, ctx := errgroup.WithContext(ctx)

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err, "Load config")
	}

	loggerSvc, err := logger.Initialize(cfg.Logger)
	if err != nil {
		log.Fatal(err, "Init logger")
	}
	ctx = loggerSvc.Zerolog().WithContext(ctx)

	jwtCfg, err := jwt.NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	jwt.Initialize(jwtCfg)

	db, err := pgx.NewPostgres(ctx, cfg.DB)
	if err != nil {
		log.Fatal(err, "Init db")
	}
	if err := db.Start(ctx, runner); err != nil {
		log.Fatal(err, "Start db")
	}

	storage := postgres.NewStorage(ctx, db)

	authSvc := auth.New(storage)
	profSvc := profile.New(storage)
	modSvc := moderation.New(storage)
	reportSvc := report.New(storage)
	playSessionSvc := session.New(storage)

	httpSrv := server.NewServer(cfg.ServerAddress, authSvc, profSvc, modSvc, reportSvc, playSessionSvc)
	httpSrv.Run(ctx, runner)

	runner.Go(func() error {
		<-ctx.Done()

		if err := db.Shutdown(ctx); err != nil {
			loggerSvc.Zerolog().Error().Err(err).Msg("Shutdown db")
			return err
		}
		return httpSrv.Shutdown(ctx)
	})

	runner.Wait()
}
