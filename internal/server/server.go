package server

import (
	"context"

	api "github.com/morning-night-dream/distributed-services-with-go/api/v1"
	"google.golang.org/grpc"
)

type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
}

type Config struct {
	CommitLog CommitLog
}

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

func NewGRPCServer(config *Config) (*grpc.Server, error) {
	gsrv := grpc.NewServer()
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
