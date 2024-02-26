// SPDX-License-Identifier: MIT
//
// Copyright (c) 2024 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package client

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/itsdevbear/bolaris/config"
	eth "github.com/itsdevbear/bolaris/engine/client/ethclient"
	"github.com/itsdevbear/bolaris/io/http"
	"github.com/itsdevbear/bolaris/io/jwt"
)

// Caller is implemented by EngineClient.
var _ Caller = (*EngineClient)(nil)

// EngineClient is a struct that holds a pointer to an Eth1Client.
type EngineClient struct {
	*eth.Eth1Client

	cfg          *config.Engine
	beaconCfg    *config.Beacon
	capabilities map[string]struct{}
	logger       log.Logger
	isConnected  atomic.Bool
	jwtSecret    *jwt.Secret
}

// New creates a new engine client EngineClient.
// It takes an Eth1Client as an argument and returns a pointer to an
// EngineClient.
func New(opts ...Option) *EngineClient {
	ec := &EngineClient{
		Eth1Client:   new(eth.Eth1Client),
		capabilities: make(map[string]struct{}),
	}

	for _, opt := range opts {
		if err := opt(ec); err != nil {
			panic(err)
		}
	}

	return ec
}

// Start starts the engine client.
func (s *EngineClient) Start(ctx context.Context) {
	// Attempt an initial connection.
	s.tryConnectionAfter(ctx, 0)

	// We will spin up the execution client connection in a
	// loop until it is connected.
	for !s.isConnected.Load() {
		// If we enter this loop, the above connection attempt failed.
		s.logger.Info(
			"Waiting for connection to execution client...",
			"engine-dial-url", s.cfg.RPCDialURL.String(),
		)
		s.tryConnectionAfter(ctx, s.cfg.RPCStartupCheckInterval)
	}

	// Exchange capabilities with the execution client.
	if _, err := s.ExchangeCapabilities(ctx); err != nil {
		s.logger.Error("failed to exchange capabilities", "err", err)
	}

	// If we reached this point, the execution client is connected so we can
	// start the jwt refresh loop.
	go s.jwtRefreshLoop(ctx)
}

// Status verifies the chain ID via JSON-RPC. By proxy
// we will also verify the connection to the execution client.
func (s *EngineClient) Status() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.RPCTimeout)
	defer cancel()
	return s.VerifyChainID(ctx)
}

// Checks the chain ID of the execution client to ensure
// it matches local parameters of what Prysm expects.
func (s *EngineClient) VerifyChainID(ctx context.Context) error {
	chainID, err := s.Client.ChainID(ctx)
	if err != nil {
		return err
	}

	if chainID.Uint64() != s.cfg.RequiredChainID {
		return fmt.Errorf(
			"wanted chain ID %d, got %d",
			s.cfg.RequiredChainID,
			chainID.Uint64(),
		)
	}

	return nil
}

// jwtRefreshLoop refreshes the JWT token for the execution client.
func (s *EngineClient) jwtRefreshLoop(ctx context.Context) {
	for {
		s.tryConnectionAfter(ctx, s.cfg.RPCJWTRefreshInterval)
	}
}

// tryConnectionAfter attempts a connection after a given interval.
func (s *EngineClient) tryConnectionAfter(
	ctx context.Context, interval time.Duration,
) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(interval):
		s.setupExecutionClientConnection(ctx)
	}
}

// setupExecutionClientConnections dials the execution client and
// ensures the chain ID is correct.
func (s *EngineClient) setupExecutionClientConnection(ctx context.Context) {
	// Dial the execution client.
	if err := s.dialExecutionRPCClient(ctx); err != nil {
		// This log gets spammy, we only log it when we first lose connection.
		if s.isConnected.Load() {
			s.logger.Error("could not dial execution client", "error", err)
		}
		s.isConnected.Store(false)
		return
	}

	// Ensure the execution client is connected to the correct chain.
	if err := s.VerifyChainID(ctx); err != nil {
		s.Client.Close()
		if strings.Contains(err.Error(), "401 Unauthorized") {
			// We always log this error as it is a critical error.
			s.logger.Error(UnauthenticatedConnectionErrorStr)
		} else if s.isConnected.Load() {
			// This log gets spammy, we only log it when we first lose
			// connection.
			s.logger.Error("could not dial execution client", "error", err)
		}

		s.isConnected.Store(false)
		return
	}

	// If we reached here the client is connected and we mark as such.
	s.isConnected.Store(true)
}

// DialExecutionRPCClient dials the execution client's RPC endpoint.
func (s *EngineClient) dialExecutionRPCClient(ctx context.Context) error {
	var (
		client *rpc.Client
		err    error
	)

	// Dial the execution client based on the URL scheme.
	switch s.cfg.RPCDialURL.Scheme {
	case "http", "https":
		client, err = rpc.DialOptions(
			ctx, s.cfg.RPCDialURL.String(), rpc.WithHeaders(
				http.NewHeaderWithJWT(s.jwtSecret)),
		)
	case "", "ipc":
		client, err = rpc.DialIPC(ctx, s.cfg.RPCDialURL.String())
	default:
		return fmt.Errorf(
			"no known transport for URL scheme %q",
			s.cfg.RPCDialURL.Scheme,
		)
	}

	// Check for an error when dialing the execution client.
	if err != nil {
		return err
	}

	s.Client = ethclient.NewClient(client)
	return nil
}
