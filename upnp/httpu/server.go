// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package httpu

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/ethulhu/helix/logger"
)

type (
	Server struct {
		Handler func(*http.Request) []Response
		mu      sync.RWMutex
		conn    net.PacketConn
		running atomic.Bool
	}

	Response map[string]string
)

func (r Response) Bytes() []byte {
	var keys []string
	for k := range r {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	fmt.Fprint(&buf, "HTTP/1.1 200 OK\r\n")
	for _, k := range keys {
		v := r[k]
		if v == "" {
			fmt.Fprintf(&buf, "%s:\r\n", k)
		} else {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprint(&buf, "\r\n")
	return buf.Bytes()
}

func (s *Server) Close() error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		return nil
	}
	return conn.Close()
}

func (s *Server) Serve(conn net.PacketConn) error {
	packet := make([]byte, 2048)
	s.mu.Lock()
	if s.running.Load() {
		s.mu.Unlock()
		return errors.New("HTTPU server is already running")
	}
	s.conn = conn
	s.running.Store(true)
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.conn = nil
		s.running.Store(false)
		s.mu.Unlock()
	}()
Loop:
	for {
		n, addr, err := conn.ReadFrom(packet)
		if err != nil {
			return fmt.Errorf("could not receive HTTPU packet: %w", err)
		}

		log, ctx := logger.FromContext(context.TODO())
		log.AddField("httpu.client", addr)

		req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(packet[:n])))
		if err != nil {
			log.WithField("packet", packet[:n])
			log.WithError(err).Warning("could not deserialize HTTPU request")
			continue
		}
		log.AddField("httpu.method", req.Method)
		log.AddField("httpu.url", req.URL)

		rsps := s.Handler(req.WithContext(ctx))
		if len(rsps) == 0 {
			log.Debug("not sending an HTTPU response")
			continue
		}

		for _, rsp := range rsps {
			if _, err := conn.WriteTo(rsp.Bytes(), addr); err != nil {
				log.WithError(err).Warning("could not send HTTPU response")
				continue Loop
			}
		}

		log.Debug("served HTTPU responses")
	}
}

func (s *Server) Running() bool {
	return s.running.Load()
}
