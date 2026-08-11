package stakingrouter

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// authHeader carries the bearer token. Staking Router's Auth0 interceptor
	// reads "authorization: Bearer <jwt>", matching the REST gateway.
	authHeader = "authorization"
	// apiKeyHeader carries the Eth-Link API key ("x-api-key"), the scheme used
	// by EthLinkService instead of a bearer token.
	apiKeyHeader = "x-api-key"
)

// authInterceptors returns unary and stream interceptors that inject the bearer
// token (from src, if any) and the api-key (if non-empty) as outgoing metadata.
// Returns (nil, nil) when there is nothing to inject, so the caller can skip
// registering interceptors entirely.
func authInterceptors(src TokenSource, apiKey string) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	if src == nil && apiKey == "" {
		return nil, nil
	}

	// attach appends the configured credentials to the outgoing context.
	attach := func(ctx context.Context) (context.Context, error) {
		if apiKey != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, apiKeyHeader, apiKey)
		}
		if src != nil {
			token, err := src.Token(ctx)
			if err != nil {
				return nil, err
			}
			if token != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, authHeader, "Bearer "+token)
			}
		}
		return ctx, nil
	}

	unary := func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx, err := attach(ctx)
		if err != nil {
			return err
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	stream := func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx, err := attach(ctx)
		if err != nil {
			return nil, err
		}
		return streamer(ctx, desc, cc, method, opts...)
	}

	return unary, stream
}
