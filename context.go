package main

import (
	"context"
	"net"
)

type contextKey struct{}

func NewContext(ctx context.Context, v any) context.Context {
	return context.WithValue(ctx, contextKey{}, v)
}

func FromContext(ctx context.Context) any {
	return ctx.Value(contextKey{})
}

type connContextKey struct{}

func NewNetConnContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, connContextKey{}, c)
}

func FromNetConnContext(ctx context.Context) (net.Conn, bool) {
	c, ok := ctx.Value(connContextKey{}).(net.Conn)
	return c, ok
}

type netAddrContextKey struct{}

func NewNetAddrContext(ctx context.Context, a net.Addr) context.Context {
	return context.WithValue(ctx, netAddrContextKey{}, a)
}

func FromNetAddrContext(ctx context.Context) (net.Addr, bool) {
	a, ok := ctx.Value(netAddrContextKey{}).(net.Addr)
	return a, ok
}
