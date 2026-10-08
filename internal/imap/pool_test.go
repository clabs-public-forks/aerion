package imap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var errNoCredentials = errors.New("no credentials")

func newTestPool(max int) *Pool {
	return NewPool(PoolConfig{
		MaxConnections: max,
		WaiterTimeout:  5 * time.Second,
	}, func(string) (*ClientConfig, error) {
		return nil, errNoCredentials
	})
}

// A connection with a nil client is unhealthy without needing a server.
func addDeadConn(p *Pool, accountID string, inUse bool) *PooledConnection {
	conn := &PooledConnection{accountID: accountID, inUse: inUse}
	p.connections[accountID] = append(p.connections[accountID], conn)
	return conn
}

func TestReleaseUnhealthyRemovesConnectionAndWakesWaiter(t *testing.T) {
	p := newTestPool(1)
	conn := addDeadConn(p, "acct", true)

	result := make(chan error, 1)
	go func() {
		_, err := p.GetConnection(context.Background(), "acct")
		result <- err
	}()

	// Wait until the request is queued behind the full pool.
	deadline := time.Now().Add(2 * time.Second)
	for {
		p.mu.Lock()
		n := len(p.waiters["acct"])
		p.mu.Unlock()
		if n == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("request never waited for a connection")
		}
		time.Sleep(5 * time.Millisecond)
	}

	p.Release(conn)

	select {
	case err := <-result:
		// The woken waiter retried and attempted a replacement connection.
		if err == nil || !strings.Contains(err.Error(), errNoCredentials.Error()) {
			t.Fatalf("GetConnection error = %v, want credentials error from replacement attempt", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter was not woken after unhealthy release")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if n := len(p.connections["acct"]); n != 0 {
		t.Fatalf("pool still tracks %d connections, want 0", n)
	}
}

func TestGetConnectionDropsIdleDeadConnections(t *testing.T) {
	p := newTestPool(1)
	addDeadConn(p, "acct", false)

	_, err := p.GetConnection(context.Background(), "acct")
	if err == nil || !strings.Contains(err.Error(), errNoCredentials.Error()) {
		t.Fatalf("GetConnection error = %v, want credentials error from new connection attempt", err)
	}
	if n := len(p.connections["acct"]); n != 0 {
		t.Fatalf("pool still tracks %d connections, want 0", n)
	}
}
