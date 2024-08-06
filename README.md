# Pool

A generic connection pool for go.

## Usage

### Installation

```
go get github.com/henrywhitaker3/pool
```

### Creating a pool

To create a pool, you connection item must implement the `Connection` interface:

```go
type Connection interface{
    Close() error
}
```

You can then create a new pool:

```go
import (
    "fmt"
    "time"

    "github.com/henrywhitaker3/pool"
)

type conn struct{}

func (c *conn) Hello() {
    fmt.Println("hello!")
}

func (c *conn) Close() error {
    return nil
}

func main() {
    pool, err := pool.New(pool.PoolOptions[*conn]{
        Connect: func() (*conn, error) {
            return &conn{}, nil
        },
        Connections: 5,
        ProbeInterval: time.Second,
    })
    if err != nil {
        panic(err)
    }

    conn, ok := pool.Next()
    if !ok {
        panic("no connection in pool")
    }
    conn.Hello()
}
```

### Use your own logger

By default, a pool will use the builtin `log` package to print to stdout, you can pass your own logger:

```go
pool.PoolOptions[*conn]{
    Logger: myLogger,
}
```

### Metrics

You can provide prometheus metrics to the pool:

```go
pool.PoolOptions[*conn]{
    Metrics: &pool.PoolMetrics{
        // The number of healthy connections in the pool
        Connections: connectionsGauge,
        // The number of errors when creating a new pool connection
        ConnectionErrors: connectionErrorCounter,
    }
}
```
