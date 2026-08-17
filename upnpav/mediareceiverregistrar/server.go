// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package mediareceiverregistrar

import (
	"context"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) IsAuthorized(_ context.Context, _ string) (int, error) {
	return 1, nil
}

func (s *Server) RegisterDevice(_ context.Context, _ []byte) ([]byte, error) {
	return []byte{}, nil
}

func (s *Server) IsValid(_ context.Context, _ string) (int, error) {
	return 1, nil
}
