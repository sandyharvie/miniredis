package miniredis

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/bsm/redeo"
)

// Miniredis is a Redis server implementation.
type Miniredis struct {
	sync.Mutex
	closed     chan struct{}
	listen     net.Listener
	info       *redeo.ServerInfo
	stringKeys map[string]string // GET/SET keys
	expire     map[string]int    // EXPIRE values
}

var errUnimplemented = errors.New("unimplemented")

// NewMiniRedis makes a new non-started Miniredis object.
func NewMiniRedis() *Miniredis {
	return &Miniredis{
		closed:     make(chan struct{}),
		stringKeys: make(map[string]string),
		expire:     make(map[string]int),
	}
}

// Run creates and Start()s a Miniredis.
func Run() (*Miniredis, error) {
	m := NewMiniRedis()
	return m, m.Start()
}

// Close shuts down a Miniredis.
func (m *Miniredis) Close() {
	m.Lock()
	defer m.Unlock()
	if m.listen == nil {
		return
	}
	if m.listen.Close() != nil {
		return
	}
	<-m.closed
	m.listen = nil
}

// Start starts a server. It listens on a random port on localhost. See also Addr().
func (m *Miniredis) Start() error {
	m.Lock()
	defer m.Unlock()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			return fmt.Errorf("failed to listen on a port: %v", err)
		}
	}
	m.listen = l
	srv := redeo.NewServer(&redeo.Config{Addr: "localhost:0"})

	m.info = srv.Info()

	commandsConnection(m, srv)
	commandsGeneric(m, srv)
	commandsString(m, srv)

	go func() {
		e := make(chan error)
		go srv.Serve(e, m.listen)
		<-e
		m.closed <- struct{}{}
	}()
	return nil
}

// Addr returns '127.0.0.1:12345'. Can be given to a Dial()
func (m *Miniredis) Addr() string {
	m.Lock()
	defer m.Unlock()
	return m.listen.Addr().String()
}

// CommandCount returns the number of processed commands.
func (m *Miniredis) CommandCount() int {
	m.Lock()
	defer m.Unlock()
	return int(m.info.TotalProcessed())
}

// CurrentConnectionCount returns the number of currently connected clients.
func (m *Miniredis) CurrentConnectionCount() int {
	m.Lock()
	defer m.Unlock()
	return m.info.ClientsLen()
}

// TotalConnectionCount returns the number of client connections since server start.
func (m *Miniredis) TotalConnectionCount() int {
	m.Lock()
	defer m.Unlock()
	return int(m.info.TotalConnections())
}

// Get returns string keys added with SET.
// This will return an empty string if the key is not set. Redis would return
// a nil.
func (m *Miniredis) Get(k string) string {
	m.Lock()
	defer m.Unlock()
	return m.stringKeys[k]
}

// Set set a string key.
func (m *Miniredis) Set(k string, v string) {
	m.Lock()
	defer m.Unlock()
	m.stringKeys[k] = v
}

// Expire value. As set by the client. 0 if not set.
func (m *Miniredis) Expire(k string) int {
	m.Lock()
	defer m.Unlock()
	return m.expire[k]
}
