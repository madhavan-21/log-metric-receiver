package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	me "bitbucket.org/minion/metrics-system/consumer/memorizer"
	pb "bitbucket.org/minion/metrics-system/protobuf/metric_protobuf"
	_ "github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"google.golang.org/protobuf/proto"
)

// MetricConsumer listens to the NSQ topic and processes incoming metrics
type MetricConsumer struct {
	nsqConsumer *nsq.Consumer
	mem         *me.Memorizer
	db          *sql.DB
}

// NewMetricConsumer initializes NSQ consumer
func NewMetricConsumer(nsqAddr, topic, channel, dbConnStr string) (*MetricConsumer, error) {
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		return nil, err
	}

	// Connect to TimescaleDB database
	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		return nil, err
	}

	// Initialize Metric slice in the consumer
	mc := &MetricConsumer{
		nsqConsumer: consumer,
		mem:         me.NewMemorizer(),
		db:          db,
	}

	consumer.AddHandler(nsq.HandlerFunc(mc.handleMessage))

	// Connect to NSQ server
	if err := consumer.ConnectToNSQD(nsqAddr); err != nil {
		return nil, err
	}

	return mc, nil
}

// handleMessage processes each metric message from NSQ
func (mc *MetricConsumer) handleMessage(message *nsq.Message) error {
	var batch pb.MetricBatch
	if err := proto.Unmarshal(message.Body, &batch); err != nil {
		log.Printf("❌ Failed to unmarshal message: %v", err)
		return err
	}

	log.Printf("📥 Received %d metrics.", len(batch.Metrics))
	for _, metric := range batch.Metrics {
		var me me.Metric
		me.Name = metric.Name
		me.Type = metric.Type
		me.Value = metric.Value
		me.Timestamp = metric.Timestamp.AsTime().Nanosecond()
		me.Tags = metric.Tags
		me.ProjectName = metric.ProjectName
		me.Hostname = metric.Hostname
		me.Os = metric.Os
		me.UniqueId = metric.UniqueId
		me.Unit = metric.Unit

		res, err := mc.mem.ContextMemorizer(me)
		if err != nil {
			fmt.Printf("�� Failed to memorize metric: %v", err.Error())
			continue
		}

		// Insert the metric into the TimescaleDB
		if err := mc.saveMetricToDB(res); err != nil {
			log.Printf("❌ Failed to store metric in DB: %v", err)
			continue
		}

		log.Printf("✅ Successfully stored metric: %v", me.Name)
	}

	return nil
}

// saveMetricToDB saves the metric into TimescaleDB
func (mc *MetricConsumer) saveMetricToDB(metric me.Metric) error {

	tagsJSON, err := json.Marshal(metric.Tags)
	if err != nil {
		log.Printf("❌ Failed to serialize tags: %v", err)
		return err
	}
	query := `
		INSERT INTO metrics (name, type, value, timestamp, tags, project_name, hostname, os, unique_id, unit)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = mc.db.Exec(query, metric.Name, metric.Type, metric.Value, metric.Timestamp, tagsJSON, metric.ProjectName, metric.Hostname, metric.Os, metric.UniqueId, metric.Unit)

	return err
}

func main() {
	nsqAddr := "127.0.0.1:4150"
	topic := "metrics_protobuf"
	channel := "metrics_channel"
	dbConnStr := "user=metrics_user password=metrics_password dbname=metrics_db sslmode=disable host=127.0.0.1 port=5433"

	// Initialize the MetricConsumer
	mc, err := NewMetricConsumer(nsqAddr, topic, channel, dbConnStr)
	if err != nil {
		log.Fatalf("❌ Failed to initialize NSQ consumer: %v", err)
	}
	log.Println("🚀 NSQ Metric Consumer is running...")

	// Graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Block and wait for a termination signal
	<-stopChan

	// Gracefully stop the consumer before exiting
	mc.nsqConsumer.Stop()
	mc.db.Close()
	log.Println("✅ NSQ Metric Consumer stopped gracefully.")
}
