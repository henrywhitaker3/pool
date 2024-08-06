package main

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
		Connections:   5,
		ProbeInterval: time.Second,
		Logger:        pool.NullLogger,
	})
	if err != nil {
		panic(err)
	}

	conn, ok := pool.Next()
	if !ok {
		panic("no connection in pool")
	}

	fmt.Printf("There are %d connections in the pool\n", pool.Size())

	conn.Hello()
}
