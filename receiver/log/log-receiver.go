package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "bitbucket.org/minion/metrics-system/protobuf/log_protobuf"

	"github.com/nsqio/go-nsq"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type LogReceiver struct {
	pb.UnimplementedLogReceiverServer
	nsqProducer *nsq.Producer
}

// NewLogReceiver initializes the gRPC receiver with an NSQ producer
func NewLogReceiver(nsqAddr string) (*LogReceiver, error) {
	producer, err := nsq.NewProducer(nsqAddr, nsq.NewConfig())
	if err != nil {
		return nil, err
	}
	return &LogReceiver{nsqProducer: producer}, nil
}

// ReceiveLogs processes incoming logs and forwards them to NSQ
func (r *LogReceiver) ReceiveLogs(ctx context.Context, req *pb.LogBatch) (*pb.LogResponse, error) {
	if len(req.Logs) == 0 {
		log.Println("⚠️ No logs received, dropping request.")
		return &pb.LogResponse{Status: "No logs received"}, nil
	}

	// Convert the protobuf message to bytes
	data, err := proto.Marshal(req)
	if err != nil {
		log.Printf("❌ Failed to serialize protobuf: %v", err)
		return nil, err
	}

	// Print incoming logs
	fmt.Println("📥 Incoming Logs:")
	for _, logEntry := range req.Logs {
		fmt.Printf(
			"[%s] %s - %s:%d %s | %s\n",
			logEntry.Level, logEntry.Timestamp.AsTime().Format("2006-01-02 15:04:05"),
			logEntry.File, logEntry.Line, logEntry.Function, logEntry.Message,
		)
	}

	// Publish Protobuf bytes to NSQ topic
	topic := "logs_protobuf"
	err = r.nsqProducer.Publish(topic, data)
	if err != nil {
		log.Printf("❌ Failed to publish to NSQ: %v", err)
		return nil, err
	}

	log.Printf("📤 Sent %d logs to NSQ topic %s", len(req.Logs), topic)
	return &pb.LogResponse{Status: "Logs processed successfully"}, nil
}

func main() {
	nsqAddr := "127.0.0.1:4150"
	receiver, err := NewLogReceiver(nsqAddr)
	if err != nil {
		log.Fatalf("❌ Failed to initialize NSQ producer: %v", err)
	}

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLogReceiverServer(grpcServer, receiver)

	log.Println("🚀 gRPC Log Receiver is running on port 50051...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Failed to serve: %v", err)
	}
}
