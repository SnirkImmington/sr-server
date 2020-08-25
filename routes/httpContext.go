package routes

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

type srContextKey int

const (
	requestIDKey        = srContextKey(0)
	requestConnectedKey = srContextKey(1)
	requestStartedKey   = srContextKey(2)
)

func withRequestID(ctx context.Context) context.Context {
	id := fmt.Sprintf("%02x", rand.Intn(256))
	return context.WithValue(ctx, requestIDKey, id)
}

func requestID(ctx context.Context) string {
	val := ctx.Value(requestIDKey)
	if val == nil {
		log.Output(2, "Attempted to get request ID from missing context")
		return "??"
	}
	return val.(string)
}

func withConnectedNow(ctx context.Context) context.Context {
	now := time.Now()
	return context.WithValue(ctx, requestConnectedKey, now)
}

func connectedAt(ctx context.Context) time.Time {
	val := ctx.Value(requestConnectedKey)
	if val == nil {
		log.Output(2, fmt.Sprintf("Unable to get request connected at: %v", ctx))
		return time.Now()
	}
	return val.(time.Time)
}

func connContext(ctx context.Context, conn net.Conn) context.Context {
	return withConnectedNow(withRequestID(ctx))
}
