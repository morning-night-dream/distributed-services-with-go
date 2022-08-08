package server

import (
	"context"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	api "github.com/morning-night-dream/distributed-services-with-go/api/v1"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

type Config struct {
	CommitLog  CommitLog
	Authorizer Authorizer
}

const (
	objectWildcard = "*"
	produceAction  = "produce"
	consumeAction  = "consume"
)

// 空のポインタにすることで、空の構造体分のメモリを確保しなくてもよく、メモリの節約になる。
var _ api.LogServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	api.UnimplementedLogServiceServer // これ入れていたら上記の var _ api.LogServiceServer = (*grpcServer)(nil)
	*Config
}

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}
	return srv, nil
}

func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		produceAction,
	); err != nil {
		return nil, err
	}
	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) ProduceStream(stream api.LogService_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		res, err := s.Produce(stream.Context(), s.toProduceRequest(req))
		if err != nil {
			return err
		}
		if err = stream.Send(s.toProduceStreamResponse(res)); err != nil {
			return err
		}
	}
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		consumeAction,
	); err != nil {
		return nil, err
	}
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}

	return &api.ConsumeResponse{Record: record}, nil
}

func (s *grpcServer) ConsumeStream(
	req *api.ConsumeStreamRequest,
	stream api.LogService_ConsumeStreamServer,
) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), s.toConsumeRequest(req))
			// この書き方はリンターに怒られそう
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}
			if err = stream.Send(s.toConsumeStreamResponse(res)); err != nil {
				return err
			}
			req.Offset++
		}
	}
}

func NewGRPCServer(config *Config, grpcOpts ...grpc.ServerOption) (*grpc.Server, error) {
	logger := zap.L().Named("server")
	zapOpts := []grpc_zap.Option{
		grpc_zap.WithDurationField(
			func(duration time.Duration) zapcore.Field {
				return zap.Int64(
					"grpc.time_ns",
					duration.Nanoseconds(),
				)
			},
		),
	}
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	err := view.Register(ocgrpc.DefaultClientViews...)
	if err != nil {
		return nil, err
	}
	grpcOpts = append(
		grpcOpts,
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_ctxtags.StreamServerInterceptor(),
				grpc_auth.StreamServerInterceptor(authenticate),
				grpc_zap.StreamServerInterceptor(logger, zapOpts...),
			),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_ctxtags.UnaryServerInterceptor(),
				grpc_auth.UnaryServerInterceptor(authenticate),
				grpc_zap.UnaryServerInterceptor(logger, zapOpts...),
			),
		),
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	)
	gsrv := grpc.NewServer(grpcOpts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	// オブジェクト指向が強すぎるのか、
	// 以下のような書き方が直感的じゃないと感じてしまう
	// gsev.RegisterService(srv) みたいな方がなんとなく分かりやすそう
	api.RegisterLogServiceServer(gsrv, srv)
	return gsrv, nil
}

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find peer info",
		).Err()
	}

	if peer.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}

// grpcの宣言を変更したので
// 以下の変換処理が必要となった

func (s *grpcServer) toProduceRequest(req *api.ProduceStreamRequest) *api.ProduceRequest {
	return &api.ProduceRequest{
		Record: req.Record,
	}
}

func (s *grpcServer) toProduceStreamResponse(req *api.ProduceResponse) *api.ProduceStreamResponse {
	return &api.ProduceStreamResponse{
		Offset: req.Offset,
	}
}

func (s *grpcServer) toConsumeRequest(req *api.ConsumeStreamRequest) *api.ConsumeRequest {
	return &api.ConsumeRequest{
		Offset: req.Offset,
	}
}

func (s *grpcServer) toConsumeStreamResponse(res *api.ConsumeResponse) *api.ConsumeStreamResponse {
	return &api.ConsumeStreamResponse{
		Record: res.Record,
	}
}
