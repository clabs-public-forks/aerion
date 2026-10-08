package imap

import (
	"bufio"
	"crypto/tls"
	"net"
	"strings"
	"testing"
	"time"
)

// TestStartTLSDefaultsServerName checks that a STARTTLS connection with no
// TLSConfig sends the configured host as SNI instead of failing the
// handshake for want of a ServerName.
func TestStartTLSDefaultsServerName(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	sni := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		r := bufio.NewReader(conn)
		_, _ = conn.Write([]byte("* OK ready\r\n"))
		line, _ := r.ReadString('\n')
		tag := strings.Fields(line)[0]
		_, _ = conn.Write([]byte(tag + " OK begin TLS\r\n"))
		srv := tls.Server(conn, &tls.Config{GetConfigForClient: func(h *tls.ClientHelloInfo) (*tls.Config, error) {
			sni <- h.ServerName
			return nil, net.ErrClosed // stop after the ClientHello
		}})
		_ = srv.Handshake()
	}()

	cfg := DefaultConfig()
	cfg.Host = "localhost"
	cfg.Port = ln.Addr().(*net.TCPAddr).Port
	cfg.Security = SecurityStartTLS
	cfg.ConnectTimeout = 5 * time.Second
	c := NewClient(cfg)
	err = c.Connect()
	if err == nil {
		t.Fatal("Connect succeeded against a server that aborts the handshake")
	}
	if strings.Contains(err.Error(), "ServerName") {
		t.Fatalf("handshake failed for want of a ServerName: %v", err)
	}
	select {
	case got := <-sni:
		if got != "localhost" {
			t.Errorf("SNI = %q, want localhost", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server never saw a ClientHello")
	}
}
