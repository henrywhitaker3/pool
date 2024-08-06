package pool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Connection interface {
	Close() error
}

type ConnectFunc[T Connection] func() (T, error)
type HealthyFunc[T Connection] func(T) bool

type PoolMetrics struct {
	// The number of connections currently in the pool
	Connections prometheus.Gauge
	// The number of errors when connecting
	ConnectionErrors prometheus.Counter
	// The number of connections retrieved from the pool
	Retrievals prometheus.Counter
}

type PoolOptions[T Connection] struct {
	// A function that is run the create a new connection
	Connect ConnectFunc[T]
	// AN optional function that is used to check the health of the connection
	Healthy HealthyFunc[T]
	// The number of connections in the pool
	Connections int
	// The time interval that health probes and pool size checking
	ProbeInterval time.Duration
	Logger        Logger

	Metrics *PoolMetrics

	// MinConnections uint
	// MaxConnections uint
}

func (p PoolOptions[T]) validate() error {
	if p.Connect == nil {
		return ErrNoConnectFunc
	}
	if p.Connections < 1 {
		return ErrInvalidConnectionCount
	}
	if p.ProbeInterval <= 0 {
		return ErrInvalidProbeInterval
	}
	return nil
}

type Pool[T Connection] struct {
	opts PoolOptions[T]

	log Logger

	pool []T
	mu   *sync.Mutex

	prober *sync.Mutex

	closer *sync.Once
	closed chan struct{}

	next uint64
}

func New[T Connection](opts PoolOptions[T]) (*Pool[T], error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	logger := opts.Logger
	if logger == nil {
		logger = DefaultLogger
	}

	pool := &Pool[T]{
		opts:   opts,
		log:    logger,
		pool:   []T{},
		mu:     &sync.Mutex{},
		prober: &sync.Mutex{},
		closer: &sync.Once{},
		closed: make(chan struct{}, 1),
		next:   0,
	}

	pool.connect()

	return pool, nil
}

// Gets the next connection from the pool. Leaves the connection
// in the pool available to other consumers.
func (p *Pool[T]) Next() (T, bool) {
	len := p.Size()
	if len == 0 {
		var empty T
		return empty, false
	}

	n := atomic.AddUint64(&p.next, 1)
	p.reportRetrieval()

	return p.pool[int(n)%len], true
}

// Gets the next connection from the pool exclusively - i.e. it is not
// available for other consumers to use.
func (p *Pool[T]) ExNext() (T, bool) {
	len := p.Size()
	if p.Size() == 0 {
		var empty T
		return empty, false
	}
	n := atomic.AddUint64(&p.next, 1)

	index := int(n) % len

	p.mu.Lock()
	conn := p.pool[index]
	p.mu.Unlock()

	p.reportRetrieval()

	p.removeFromPool(index)
	return conn, true
}

// Return a ExNext connection to the pool
func (p *Pool[T]) Return(c T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool = append(p.pool, c)
}

func (p *Pool[T]) Closed() <-chan struct{} {
	return p.closed
}

func (p *Pool[T]) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pool)
}

func (p *Pool[T]) Close() {
	p.closer.Do(func() {
		p.log.Debugf("closing connection pool")
		p.closed <- struct{}{}

		len := p.Size()
		for range len {
			if err := p.pool[0].Close(); err != nil {
				p.log.Errorf("failed to close connection: %v", err)
			}
			p.removeFromPool(0)
		}
	})
}

func (p *Pool[T]) connect() {
	p.log.Debugf("creating connection pool with %d connections", p.opts.Connections)

	p.probe()

	tick := time.NewTicker(p.opts.ProbeInterval)
	go func() {
		defer tick.Stop()
		for {
			select {
			case <-p.Closed():
				return
			case <-tick.C:
				go p.probe()
			}
		}
	}()
}

func (p *Pool[T]) reportSize(size int) {
	if p.opts.Metrics != nil {
		if p.opts.Metrics.Connections != nil {
			p.opts.Metrics.Connections.Set(float64(size))
		}
	}
}

func (p *Pool[T]) reportError() {
	if p.opts.Metrics != nil {
		if p.opts.Metrics.ConnectionErrors != nil {
			p.opts.Metrics.ConnectionErrors.Inc()
		}
	}
}

func (p *Pool[T]) reportRetrieval() {
	if p.opts.Metrics != nil {
		if p.opts.Metrics.Retrievals != nil {
			p.opts.Metrics.Retrievals.Inc()
		}
	}
}

func (p *Pool[T]) probe() {
	p.prober.Lock()
	defer p.prober.Unlock()

	p.log.Debugf("probing connection pool")
	select {
	case <-p.Closed():
		return
	default:
		len := p.Size()
		p.reportSize(len)
		if len == p.opts.Connections {
			return
		}

		// Check the health of all connections in the pool
		if p.opts.Healthy != nil {
			removed := 0
			for i := range len {
				if !p.opts.Healthy(p.pool[i-removed]) {
					if err := p.pool[i-removed].Close(); err != nil {
						p.log.Errorf("failed to close connection: %v", err)
					}
					p.log.Errorf("removing unhealthy connection from pool")
					p.reportError()
					p.removeFromPool(i - removed)
					removed++
				}
			}
		}

		// Add in any missing connections to the pool
		diff := p.opts.Connections - len
		for range diff {
			conn, err := p.opts.Connect()
			if err != nil {
				p.log.Errorf("failed to create connection: %v", err)
				p.reportError()
				continue
			}
			p.Return(conn)
		}
	}
}

func (p *Pool[T]) removeFromPool(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool = append(p.pool[:index], p.pool[index+1:]...)
}
