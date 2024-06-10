package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/day0ops/ext-proc-service-aggregation/pkg/config"
	"github.com/day0ops/ext-proc-service-aggregation/pkg/server"
	"github.com/day0ops/ext-proc-service-aggregation/pkg/version"
	service_ext_proc_v3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var (
	grpcport = flag.String("grpcport", ":18080", "grpcport")
)

func main() {
	os.Exit(start())
}

func start() int {
	log, err := createLogger()
	if err != nil {
		fmt.Println("error setting up the logger:", err)
		return 1
	}
	log = log.With(zap.String("release", version.HumanVersion))
	defer func() {
		// If we cannot sync, there's probably something wrong with outputting logs,
		// so we probably cannot write using fmt.Println either.
		// Hence, ignoring the error for now.
		_ = log.Sync()
	}()

	flag.Parse()

	lis, err := net.Listen("tcp", *grpcport)
	if err != nil {
		log.Error("failed to listen: %v", zap.Error(err))
		return 1
	}

	sopts := []grpc.ServerOption{grpc.MaxConcurrentStreams(1000)}
	s := grpc.NewServer(sopts...)

	// register server
	service_ext_proc_v3.RegisterExternalProcessorServer(s, &server.Server{Log: log})

	// register for health checks
	grpc_health_v1.RegisterHealthServer(s, &server.HealthServer{Log: log})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		log.Info("starting gRPC server on port", zap.String("port", *grpcport))
		if err := s.Serve(lis); err != nil {
			log.Info("error starting server", zap.Error(err))
			return err
		}
		return nil
	})

	<-ctx.Done()

	eg.Go(func() error {
		log.Info("gracefully stopping gRPC server")
		s.GracefulStop()
		return nil
	})

	if err := eg.Wait(); err != nil {
		return 1
	}
	return 0
}

func createLogger() (*zap.Logger, error) {
	encoder := zap.NewProductionEncoderConfig()
	level := zap.NewAtomicLevelAt(getLevelLogger(config.LogLevel))

	zapConfig := zap.NewProductionConfig()
	zapConfig.EncoderConfig = encoder
	zapConfig.Level = level
	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.ErrorOutputPaths = []string{"stderr"}
	return zapConfig.Build()
}

func getLevelLogger(level string) zapcore.Level {
	if level == "debug" {
		return zap.DebugLevel
	}
	return zap.InfoLevel
}
