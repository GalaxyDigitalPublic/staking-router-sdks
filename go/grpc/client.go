// Package stakingrouter is the public gRPC client for Alluvial's Staking Router.
//
// It wraps the generated service clients with a single constructor that handles
// TLS and per-RPC bearer-token (JWT) authentication, so consumers don't have to
// wire up grpc.NewClient, transport credentials, and auth interceptors by hand.
//
// This file is maintained by hand in the staking-router repository
// (sdk/_grpc/client.go) and copied verbatim into this SDK at generation
// time. Do not edit it in the published SDK repo — changes there will be
// overwritten on the next sync.
package stakingrouter

import (
	"context"
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	ethereumv1 "github.com/GalaxyDigitalPublic/staking-router-sdks/go/grpc/gen/public/ethereum/v1"
	ethlinkv1 "github.com/GalaxyDigitalPublic/staking-router-sdks/go/grpc/gen/public/ethlink/v1"
	solanav1 "github.com/GalaxyDigitalPublic/staking-router-sdks/go/grpc/gen/public/solana/v1"
	phoenixv0 "github.com/GalaxyDigitalPublic/staking-router-sdks/go/grpc/gen/public/v0/ethereum"
	publicv1 "github.com/GalaxyDigitalPublic/staking-router-sdks/go/grpc/gen/public/v1"
)

// Client is a connected Staking Router gRPC client. It exposes one typed client
// per public service. Construct it with New and release it with Close.
//
// The zero value is not usable; always obtain a Client from New.
type Client struct {
	conn *grpc.ClientConn

	// Ethereum is the primary Ethereum staking API (StakingRouterService).
	Ethereum ethereumv1.StakingRouterServiceClient
	// Solana is the Solana staking API (SolanaStakingRouterService).
	Solana solanav1.SolanaStakingRouterServiceClient
	// Webhooks manages webhook subscriptions (WebhookService).
	Webhooks publicv1.WebhookServiceClient
	// EthLink is the eth-link API (EthLinkService).
	EthLink ethlinkv1.EthLinkServiceClient
	// Phoenix is the legacy v0 Ethereum API (PhoenixService).
	Phoenix phoenixv0.PhoenixServiceClient
}

// config holds resolved dial options. Mutated only through Option funcs.
type config struct {
	plaintext   bool
	token       string
	tokenSource TokenSource
	apiKey      string
	dialOpts    []grpc.DialOption
}

// TokenSource returns a bearer token for the next RPC. Implement this when the
// token rotates (e.g. an OAuth2 client-credentials token that expires); the
// returned value is sent as "authorization: Bearer <token>" on every call.
//
// Return an empty string to send no Authorization header for that call.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// Option configures the Client at construction time.
type Option func(*config)

// WithToken authenticates every RPC with a static bearer token (JWT).
// Use WithTokenSource instead when the token rotates.
func WithToken(token string) Option {
	return func(c *config) { c.token = token }
}

// WithTokenSource authenticates every RPC with a dynamically sourced bearer
// token. Takes precedence over WithToken if both are set.
func WithTokenSource(src TokenSource) Option {
	return func(c *config) { c.tokenSource = src }
}

// WithAPIKey authenticates every RPC with a static x-api-key header. This is the
// scheme used by the Eth-Link API (EthLinkService); the Ethereum, Solana, and
// Webhook services use bearer-token auth (WithToken / WithTokenSource) instead.
// Both may be set on one Client — each RPC's auth is checked server-side.
func WithAPIKey(apiKey string) Option {
	return func(c *config) { c.apiKey = apiKey }
}

// WithPlaintext disables TLS. Use only for local development against a plaintext
// gRPC endpoint (e.g. localhost:9090 with no TLS terminator). Production
// endpoints terminate TLS and must not use this.
func WithPlaintext() Option {
	return func(c *config) { c.plaintext = true }
}

// WithDialOption appends a raw grpc.DialOption, for advanced needs (custom
// interceptors, keepalive, message-size limits) not covered by the helpers above.
func WithDialOption(opts ...grpc.DialOption) Option {
	return func(c *config) { c.dialOpts = append(c.dialOpts, opts...) }
}

// New dials addr (e.g. "api.staking.alluvial.finance:443") and returns a Client
// exposing every public service. The underlying connection is lazy — no I/O
// happens until the first RPC — so New itself does not fail on an unreachable
// server. Call Close when done.
//
// By default the connection uses TLS with the system certificate pool. Pass
// WithToken or WithTokenSource to authenticate; pass WithPlaintext only for
// local development.
func New(addr string, opts ...Option) (*Client, error) {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}

	var transport grpc.DialOption
	if cfg.plaintext {
		transport = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		transport = grpc.WithTransportCredentials(
			credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}),
		)
	}

	dialOpts := []grpc.DialOption{transport}
	if unary, stream := authInterceptors(resolveTokenSource(&cfg), cfg.apiKey); unary != nil {
		// Chained, not WithUnaryInterceptor/WithStreamInterceptor: those set the
		// single-interceptor slot, so a caller passing their own via WithDialOption
		// would replace the SDK's auth interceptor and drop the credentials
		// metadata. The chained variants compose instead, so auth always runs.
		dialOpts = append(dialOpts,
			grpc.WithChainUnaryInterceptor(unary),
			grpc.WithChainStreamInterceptor(stream),
		)
	}
	dialOpts = append(dialOpts, cfg.dialOpts...)

	conn, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:     conn,
		Ethereum: ethereumv1.NewStakingRouterServiceClient(conn),
		Solana:   solanav1.NewSolanaStakingRouterServiceClient(conn),
		Webhooks: publicv1.NewWebhookServiceClient(conn),
		EthLink:  ethlinkv1.NewEthLinkServiceClient(conn),
		Phoenix:  phoenixv0.NewPhoenixServiceClient(conn),
	}, nil
}

// Conn exposes the underlying connection for advanced use (e.g. constructing a
// service client not yet surfaced as a field). Most callers should not need it.
func (c *Client) Conn() *grpc.ClientConn { return c.conn }

// Close releases the underlying connection.
func (c *Client) Close() error { return c.conn.Close() }

// resolveTokenSource returns the effective TokenSource, or nil if the client is
// unauthenticated. A dynamic WithTokenSource wins over a static WithToken.
func resolveTokenSource(cfg *config) TokenSource {
	if cfg.tokenSource != nil {
		return cfg.tokenSource
	}
	if cfg.token != "" {
		return staticToken(cfg.token)
	}
	return nil
}

// staticToken is a TokenSource that always returns the same token.
type staticToken string

func (t staticToken) Token(context.Context) (string, error) { return string(t), nil }
