package grpcserver

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TrustedSubnetInterceptor(cidr string) grpc.UnaryServerInterceptor {
	if strings.TrimSpace(cidr) == "" {
		return func(
			ctx context.Context,
			req any,
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (any, error) {
			return handler(ctx, req)
		}
	}

	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return func(
			ctx context.Context,
			req any,
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (any, error) {
			return nil, status.Error(codes.Internal, "invalid trusted subnet")
		}
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing metadata")
		}

		values := md.Get("x-real-ip")
		if len(values) == 0 {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip")
		}

		ip := net.ParseIP(strings.TrimSpace(values[0]))
		if ip == nil || !network.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "ip is not trusted")
		}

		return handler(ctx, req)
	}
}
