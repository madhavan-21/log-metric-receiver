package main

import (
	"context"
	"log"
	"net"

	pb "bitbucket.org/minion/metrics-system/protobuf/metric_protobuf"
	"github.com/nsqio/go-nsq"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type MetricReceiver struct {
	pb.UnimplementedMetricReceiverServer
	nsqProducer *nsq.Producer
}

// NewMetricReceiver initializes NSQ producer
func NewMetricReceiver(nsqAddr string) (*MetricReceiver, error) {
	producer, err := nsq.NewProducer(nsqAddr, nsq.NewConfig())
	if err != nil {
		return nil, err
	}
	return &MetricReceiver{nsqProducer: producer}, nil
}

// ReceiveMetrics processes incoming metrics and sends them as Protobuf to NSQ
func (r *MetricReceiver) ReceiveMetrics(ctx context.Context, req *pb.MetricBatch) (*pb.ReceiveResponse, error) {
	if len(req.Metrics) == 0 {
		log.Println("⚠️ No metrics received, dropping request.")
		return &pb.ReceiveResponse{Status: "No metrics received", ReceivedCount: 0}, nil
	}

	// Convert the protobuf message to bytes
	data, err := proto.Marshal(req)
	if err != nil {
		log.Printf("❌ Failed to serialize protobuf: %v", err)
		return nil, err
	}

	// Publish Protobuf bytes to NSQ topic
	topic := "metrics_protobuf"
	err = r.nsqProducer.Publish(topic, data)
	if err != nil {
		log.Printf("❌ Failed to publish to NSQ: %v", err)
		return nil, err
	}

	log.Printf("📤 Sent %d metrics to NSQ topic %s", len(req.Metrics), topic)
	return &pb.ReceiveResponse{Status: "Metrics processed successfully", ReceivedCount: int32(len(req.Metrics))}, nil
}

func main() {
	nsqAddr := "127.0.0.1:4150"
	receiver, err := NewMetricReceiver(nsqAddr)
	if err != nil {
		log.Fatalf("❌ Failed to initialize NSQ producer: %v", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMetricReceiverServer(grpcServer, receiver)

	log.Println("🚀 gRPC Metric Receiver is running on port 50051...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Failed to serve: %v", err)
	}
}
