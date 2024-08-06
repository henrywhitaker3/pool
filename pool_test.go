package pool

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type dummyConn struct {
	failing *bool

	empty string
}

func (d *dummyConn) Healthy() bool {
	return !*d.failing
}

func (d *dummyConn) Close() error {
	return nil
}

func newDummy(failing bool) *dummyConn {
	return &dummyConn{failing: &failing, empty: "no"}
}

func TestItInitsPool(t *testing.T) {
	type test struct {
		name        string
		connect     ConnectFunc[*dummyConn]
		healthy     HealthyFunc[*dummyConn]
		connections int
		interval    time.Duration
		err         error
	}

	tcs := []test{
		{
			name:    "errors with no connectfunc",
			connect: nil,
			healthy: func(c *dummyConn) bool {
				return true
			},
			connections: 1,
			interval:    time.Millisecond,
			err:         ErrNoConnectFunc,
		},
		{
			name: "errors with 0 conns",
			connect: func() (*dummyConn, error) {
				return newDummy(false), nil
			},
			healthy: func(c *dummyConn) bool {
				return true
			},
			connections: 0,
			interval:    time.Millisecond,
			err:         ErrInvalidConnectionCount,
		},
		{
			name: "errors with -ve conns",
			connect: func() (*dummyConn, error) {
				return newDummy(false), nil
			},
			healthy: func(c *dummyConn) bool {
				return true
			},
			connections: -1,
			interval:    time.Millisecond,
			err:         ErrInvalidConnectionCount,
		},
		{
			name: "creates a pool with a healthy func",
			connect: func() (*dummyConn, error) {
				return newDummy(false), nil
			},
			healthy: func(c *dummyConn) bool {
				return true
			},
			connections: 1,
			interval:    time.Millisecond,
			err:         nil,
		},
		{
			name: "creates a pool without a healthy func",
			connect: func() (*dummyConn, error) {
				return newDummy(false), nil
			},
			connections: 1,
			interval:    time.Millisecond,
			err:         nil,
		},
	}

	for _, c := range tcs {
		t.Run(c.name, func(t *testing.T) {
			pool, err := NewPool(PoolOptions[*dummyConn]{
				Connect:       c.connect,
				Healthy:       c.healthy,
				Connections:   c.connections,
				ProbeInterval: c.interval,
			})
			if err == nil {
				defer pool.Close()
			}
			require.ErrorIs(t, err, c.err)
		})
	}
}

func TestItReturnsAConnection(t *testing.T) {
	pool, err := NewPool(PoolOptions[*dummyConn]{
		Connect: func() (*dummyConn, error) {
			return newDummy(false), nil
		},
		Connections:   2,
		ProbeInterval: time.Millisecond,
	})
	require.Nil(t, err)
	defer pool.Close()

	// Wait for it to connect
	time.Sleep(time.Millisecond)

	require.Nil(t, err)
	require.Equal(t, 2, pool.Size())

	conn, ok := pool.Next()
	require.True(t, ok)
	requireConn(t, conn)
}

func TestItExlusivelyReturnsAConnection(t *testing.T) {
	pool, err := NewPool(PoolOptions[*dummyConn]{
		Connect: func() (*dummyConn, error) {
			return newDummy(false), nil
		},
		Connections:   2,
		ProbeInterval: time.Millisecond,
	})
	require.Nil(t, err)
	defer pool.Close()

	// Wait for it to connect
	time.Sleep(time.Millisecond)

	require.Nil(t, err)
	require.Equal(t, 2, pool.Size())

	conn, ok := pool.ExNext()
	require.True(t, ok)
	requireConn(t, conn)

	require.Equal(t, 1, pool.Size())
}

func requireConn(t *testing.T, c *dummyConn) {
	require.NotNil(t, c)
	require.False(t, reflect.DeepEqual(*c, dummyConn{}))
}
