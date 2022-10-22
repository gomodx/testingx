package docker

import (
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/pkg/errors"
)

func NewDockerPool() (pool *dockertest.Pool, err error) {
	pool, err = dockertest.NewPool("")
	if err != nil {
		err = errors.Wrap(err, "failed to initialize pooled docker connection")
	}

	// the pool should only wait for this long until its considered unhealthy
	pool.MaxWait = 120 * time.Second
	return
}
