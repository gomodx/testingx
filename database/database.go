package database

import (
	"context"
	"fmt"
	docker2 "github.com/sourcec0de/testingx/docker"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/samber/lo"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/pkg/errors"
)

const PostgresTestPort = "5432/tcp"

// PostgresTestInstance is a container for pooled docker resources
// Use its constructor to create an ephemeral container which can be used to perform real database integration tests against
type PostgresTestInstance struct {
	DBPool         *pgxpool.Pool
	DockerPool     *dockertest.Pool
	DockerInstance *dockertest.Resource
	DatabaseURL    string
}

func (p PostgresTestInstance) Cleanup() error {
	return p.DockerPool.Purge(p.DockerInstance)
}

type NewPostgresTestInstanceParams struct {
	// Pool overrides the *dockertest.Pool to use (one is created if this is nil)
	Pool *dockertest.Pool

	// DatabaseName is the name of the database to use injected as an environment variable (defaults to "postgres")
	DatabaseName string

	// DatabaseUsername is the username to use injected as an environment variable (defaults to "postgres")
	DatabaseUsername string

	// DatabasePassword is the password to use injected as an environment variable (defaults to "postgres")
	DatabasePassword string

	// ContainerRepository is the repository to use (defaults to "postgres")
	ContainerRepository string

	// ContainerTag is the tag to use (defaults to "14")
	ContainerTag string

	// ContainerEnv is a map of environment variables to set in the container (can override DatabaseName, DatabaseUsername and DatabasePassword)
	ContainerEnv map[string]string

	// ContainerExpiration is the number of seconds after which the container will be automatically removed
	ContainerExpiration int

	// ContainerHostPortBindings is a map of container port to a list of host ports to bind
	ContainerHostPortBindings map[docker.Port][]docker.PortBinding

	// ContainerCmd is the command to run in the container (defaults to "postgres -D /var/lib/postgresql/data")
	ContainerCmd []string

	// ContainerAutoRemove is whether the container should be automatically removed after the expiration (defaults to true)
	ContainerDisableAutoRemove bool
}

func DefaultPostgresTestInstnaceParams() *NewPostgresTestInstanceParams {
	return &NewPostgresTestInstanceParams{
		DatabaseName:              "postgres",
		DatabaseUsername:          "postgres",
		DatabasePassword:          "postgres",
		ContainerRepository:       "postgres",
		ContainerTag:              "14",
		ContainerExpiration:       30,
		ContainerEnv:              map[string]string{},
		ContainerHostPortBindings: map[docker.Port][]docker.PortBinding{},
	}
}

func setPostgresTestInstanceDefaults(params *NewPostgresTestInstanceParams) (err error) {
	if params.Pool == nil {
		if params.Pool, err = docker2.NewDockerPool(); err != nil {
			return
		}
	}

	if params.DatabaseName == "" {
		params.DatabaseName = "postgres"
	}

	if params.DatabaseUsername == "" {
		params.DatabaseUsername = "postgres"
	}

	if params.DatabasePassword == "" {
		params.DatabasePassword = "postgres"
	}

	if params.ContainerRepository == "" {
		params.ContainerRepository = "postgres"
	}

	if params.ContainerTag == "" {
		params.ContainerTag = "14"
	}

	if params.ContainerExpiration == 0 {
		params.ContainerExpiration = 120
	}

	return
}

func NewPostgresTestInstance(
	ctx context.Context,
	params *NewPostgresTestInstanceParams,
) (pgi *PostgresTestInstance, err error) {
	pgi = new(PostgresTestInstance)

	if ctx == nil {
		ctx = context.Background()
	}

	if params == nil {
		params = DefaultPostgresTestInstnaceParams()
	}

	if err = setPostgresTestInstanceDefaults(params); err != nil {
		return
	}

	pool := params.Pool
	pool.MaxWait = time.Second * 60

	pgi.DockerPool = pool

	env := lo.MapToSlice(params.ContainerEnv, func(k string, v string) string {
		return fmt.Sprintf("%s=%s", k, v)
	})

	// pulls an image, creates a container based on it and runs it
	pgi.DockerInstance, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository:   params.ContainerRepository,
		Tag:          params.ContainerTag,
		PortBindings: params.ContainerHostPortBindings,
		Cmd:          params.ContainerCmd,
		Env: append([]string{
			fmt.Sprintf("POSTGRES_DB=%s", params.DatabaseName),
			fmt.Sprintf("POSTGRES_USER=%s", params.DatabaseUsername),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", params.DatabasePassword),
		}, env...),
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = params.ContainerDisableAutoRemove != true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = errors.Wrap(err, "could not start resource")
		return
	}

	if params.ContainerExpiration > -1 {
		if err = pgi.DockerInstance.Expire(uint(params.ContainerExpiration)); err != nil {
			err = errors.Wrap(err, "failed to set container expiration")
			return
		}
	}

	hostAndPort := pgi.DockerInstance.GetHostPort("5432/tcp")
	pgi.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		params.DatabaseUsername,
		params.DatabasePassword,
		hostAndPort,
		params.DatabaseName)

	pgConfig, err := pgxpool.ParseConfig(pgi.DatabaseURL)
	if err != nil {
		return
	}

	pgConfig.LazyConnect = true

	pgi.DBPool, err = pgxpool.ConnectConfig(ctx, pgConfig)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err = pool.Retry(newPostgresRetryHandler(ctx, pgi.DBPool)); err != nil {
		err = errors.Wrap(err, "could not connect to database")
		return
	}

	return
}

// NewDebuggablePostgresTestInstance returns a postgres test instance with a container expiration of 15m (defaults to static port binding of host(5433) -> container(5432))
func NewDebuggablePostgresTestInstance(hostPort string) (*PostgresTestInstance, error) {
	if hostPort == "" {
		hostPort = "5433"
	}
	return NewPostgresTestInstance(nil, &NewPostgresTestInstanceParams{
		ContainerExpiration: 15 * 60,
		ContainerHostPortBindings: map[docker.Port][]docker.PortBinding{
			PostgresTestPort: {
				{
					HostIP:   "0.0.0.0",
					HostPort: hostPort,
				},
			},
		},
	})
}

func newPostgresRetryHandler(
	ctx context.Context,
	pool *pgxpool.Pool,
) func() (err error) {
	return func() (err error) {
		return pool.Ping(ctx)
	}
}
