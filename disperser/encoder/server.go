package encoder

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/0glabs/0g-data-avail/common"
	"github.com/0glabs/0g-data-avail/common/healthcheck"
	"github.com/0glabs/0g-data-avail/core"
	"github.com/0glabs/0g-data-avail/disperser"
	pb "github.com/0glabs/0g-data-avail/disperser/api/grpc/encoder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// TODO: Add EncodeMetrics
type Server struct {
	pb.UnimplementedEncoderServer

	config      ServerConfig
	logger      common.Logger
	coreEncoder core.Encoder
	metrics     *Metrics
	close       func()

	runningRequests chan struct{}
	requestPool     chan struct{}
}

func NewServer(config ServerConfig, logger common.Logger, coreEncoder core.Encoder, metrics *Metrics) *Server {
	return &Server{
		config:      config,
		logger:      logger,
		coreEncoder: coreEncoder,
		metrics:     metrics,

		runningRequests: make(chan struct{}, config.MaxConcurrentRequests),
		requestPool:     make(chan struct{}, config.RequestPoolSize),
	}
}

func (s *Server) EncodeBlob(ctx context.Context, req *pb.EncodeBlobRequest) (*pb.EncodeBlobReply, error) {
	select {
	case s.requestPool <- struct{}{}:
	default:
		s.metrics.IncrementRateLimitedBlobRequestNum()
		s.logger.Warn("rate limiting as request pool is full", "requestPoolSize", s.config.RequestPoolSize, "maxConcurrentRequests", s.config.MaxConcurrentRequests)
		return nil, fmt.Errorf("too many requests")
	}
	s.runningRequests <- struct{}{}
	defer s.popRequest()

	if ctx.Err() != nil {
		s.metrics.IncrementCanceledBlobRequestNum()
		return nil, ctx.Err()
	}

	reply, err := s.handleEncoding(ctx, req)
	if err != nil {
		s.metrics.IncrementFailedBlobRequestNum()
	} else {
		s.metrics.IncrementSuccessfulBlobRequestNum()
	}
	return reply, err
}

func (s *Server) popRequest() {
	<-s.requestPool
	<-s.runningRequests
}

func (s *Server) handleEncoding(ctx context.Context, req *pb.EncodeBlobRequest) (*pb.EncodeBlobReply, error) {
	begin := time.Now()

	// Convert to core EncodingParams
	var encodingParams = core.EncodingParams{
		ChunkLength: uint(req.EncodingParams.ChunkLength),
		NumChunks:   uint(req.EncodingParams.NumChunks),
	}

	commits, chunks, err := s.coreEncoder.Encode(req.Data, encodingParams)

	if err != nil {
		return nil, err
	}

	encodingTime := time.Since(begin)

	commitData, err := commits.Commitment.Serialize()
	if err != nil {
		return nil, err
	}

	lengthProofData, err := commits.LengthProof.Serialize()
	if err != nil {
		return nil, err
	}

	var chunksData [][]byte

	for _, chunk := range chunks {
		chunkSerialized, err := chunk.Serialize()
		if err != nil {
			return nil, err
		}
		// perform an operation
		chunksData = append(chunksData, chunkSerialized)
	}

	totalTime := time.Since(begin)
	s.metrics.TakeLatency(encodingTime, totalTime)

	return &pb.EncodeBlobReply{
		Commitment: &pb.BlobCommitment{
			Commitment:  commitData,
			LengthProof: lengthProofData,
			Length:      uint32(commits.Length),
		},
		Chunks: chunksData,
	}, nil
}

func (s *Server) Start() error {
	s.logger.Trace("Entering Start function...")
	defer s.logger.Trace("Exiting Start function...")

	// Serve grpc requests
	addr := fmt.Sprintf("%s:%s", disperser.Localhost, s.config.GrpcPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Could not start tcp listener: %v", err)
	}

	opt := grpc.MaxRecvMsgSize(1024 * 1024 * 300) // 300 MiB
	gs := grpc.NewServer(opt)
	reflection.Register(gs)
	pb.RegisterEncoderServer(gs, s)

	// Register Server for Health Checks
	healthcheck.RegisterHealthServer(gs)

	s.close = func() {
		err := listener.Close()
		if err != nil {
			log.Printf("failed to close listener: %v", err)
		}
		gs.GracefulStop()
	}

	s.logger.Info("port", s.config.GrpcPort, "address", listener.Addr().String(), "GRPC Listening")
	return gs.Serve(listener)
}

func (s *Server) Close() {
	if s.close == nil {
		return
	}
	s.close()
}
