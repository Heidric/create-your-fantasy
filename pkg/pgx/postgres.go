package pgx

import (
	"context"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

const ProviderName = "pgx"

type Postgres struct {
	conn               *sqlx.DB
	config             *Config
	queueWorkerStopped bool
	shuttingDown       bool
	watcherStopped     bool
}

func NewPostgres(ctx context.Context, config *Config) (*Postgres, error) {
	if config == nil {
		return nil, errors.New("invalid passed options pointer")
	}

	p := &Postgres{
		config:             config.SetDefault(),
		queueWorkerStopped: true,
	}

	return p, nil
}

func (p *Postgres) GetConn() *sqlx.DB {
	return p.conn
}

func (p *Postgres) GetConfig() *Config {
	return p.config
}

// Start starts connection workers and connection procedure itself.
func (p *Postgres) Start(ctx context.Context, errorGroup *errgroup.Group) error {
	logger := p.GetLogger(ctx)

	if p.conn != nil {
		return nil
	}
	logger.Info().Msg("establishing connection...")
	// Connect to database.
	var err error
	p.conn, err = sqlx.ConnectContext(ctx, ProviderName, p.config.DSN)
	if err != nil {
		return errors.Wrap(err, "connect to entity")
	}
	logger.Info().Msg("connection established")

	// Write connection pooling options.
	p.SetConnPoolLifetime(p.config.MaxConnectionLifetime)
	p.SetConnPoolLimits(p.config.MaxIdleConnections, p.config.MaxOpenedConnections)

	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if p.IsWatcherStopped() {
		p.SetWatcher(false)
		errorGroup.Go(func() error {
			return p.startWatcher(ctx)
		})
	}

	return nil
}

func (p *Postgres) GetLogger(ctx context.Context) *zerolog.Logger {
	logger := zerolog.Ctx(ctx).With().Logger()
	logger = logger.With().Str("name", "pgx").Logger()

	return &logger
}

// SetConnPoolLifetime sets connection lifetime.
func (p *Postgres) SetConnPoolLifetime(connMaxLifetime time.Duration) {
	// First - set passed data in connection options.
	p.config.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if p.conn != nil {
		p.conn.SetConnMaxLifetime(connMaxLifetime)
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (p *Postgres) SetConnPoolLimits(maxIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	p.config.MaxIdleConnections = maxIdleConnections
	p.config.MaxOpenedConnections = maxOpenedConnections

	// If connection already established - tweak it.
	if p.conn != nil {
		p.conn.SetMaxIdleConns(maxIdleConnections)
		p.conn.SetMaxOpenConns(maxOpenedConnections)
	}
}

// Connection watcher goroutine entrypoint.
func (p *Postgres) startWatcher(ctx context.Context) error {
	p.GetLogger(ctx).Info().Msg("starting connection watcher")

	for {
		select {
		case <-ctx.Done():
			p.GetLogger(ctx).Info().Msg("connection watcher stopped")
			p.SetWatcher(true)
			return ctx.Err()
		default:
			if err := p.Ping(ctx); err != nil {
				p.GetLogger(ctx).Error().Err(err).Msg("connection lost")
			}
		}
		time.Sleep(p.config.Timeout)
	}
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (p *Postgres) Shutdown(ctx context.Context) error {
	p.GetLogger(ctx).Info().Msg("shutting down")
	p.SetShuttingDown(true)

	if p.config.StartWatcher {
		for {
			if p.queueWorkerStopped && p.IsWatcherStopped() {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	} else if err := p.shutdown(ctx); err != nil {
		return errors.Wrapf(err, "shutdown %q", p.GetFullName())
	}

	p.GetLogger(ctx).Info().Msg("shut down")
	return nil
}

func (p *Postgres) shutdown(ctx context.Context) error {
	if p.conn == nil {
		return nil
	}
	p.GetLogger(ctx).Info().Msg("closing connection...")

	if err := p.conn.Close(); err != nil {
		return errors.Wrap(err, "failed to close connection")
	}

	p.conn = nil

	return nil
}

// Ping is pinging connection if it's alive (or we think so).
func (p *Postgres) Ping(ctx context.Context) error {
	if p.conn == nil {
		return nil
	}

	if err := p.conn.PingContext(ctx); err != nil {
		return errors.Wrap(err, "ping connection")
	}

	return nil
}

func (p *Postgres) IsShuttingDown() bool {
	return p.shuttingDown
}

func (p *Postgres) SetShuttingDown(v bool) {
	p.shuttingDown = v
}

func (p *Postgres) IsWatcherStopped() bool {
	return p.watcherStopped
}

func (p *Postgres) SetWatcher(v bool) {
	p.watcherStopped = v
}

func (p *Postgres) GetFullName() string {
	return "postgres"
}
