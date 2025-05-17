package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	pb "bitbucket.org/minion/metrics-system/protobuf/log_protobuf"
	"github.com/nsqio/go-nsq"
	"google.golang.org/protobuf/proto"
)

const (
	lokiURL     = "http://localhost:3100/loki/api/v1/push" // Loki endpoint
	quickwitURL = "http://localhost:7280/api/v1/ingest"    // Quickwit endpoint (if needed)
	useLoki     = true                                     // Change to false for Quickwit
)

// LogConsumer handles consuming logs from NSQ and forwarding them
type LogConsumer struct {
	consumer *nsq.Consumer
}

// NewLogConsumer initializes the NSQ consumer
func NewLogConsumer(nsqAddr, topic, channel string) (*LogConsumer, error) {
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		return nil, err
	}

	lc := &LogConsumer{consumer: consumer}
	consumer.AddHandler(nsq.HandlerFunc(lc.handleMessage))

	// Connect to NSQD or NSQLookupd (recommended)
	err = consumer.ConnectToNSQD(nsqAddr)
	if err != nil {
		log.Printf("⚠️  Failed to connect to NSQD: %v", err)
		log.Println("🔄 Trying NSQLookupd instead...")
		err = consumer.ConnectToNSQLookupd("127.0.0.1:4161")
		if err != nil {
			return nil, fmt.Errorf("failed to connect to NSQ: %v", err)
		}
	}
	return lc, nil
}

// handleMessage processes the incoming NSQ message
func (lc *LogConsumer) handleMessage(msg *nsq.Message) error {
	var logBatch pb.LogBatch
	if err := proto.Unmarshal(msg.Body, &logBatch); err != nil {
		log.Printf("❌ Failed to deserialize log batch: %v", err)
		return err
	}

	for _, logEntry := range logBatch.Logs {
		// Format log entry
		formattedLog := fmt.Sprintf("[%s] %s - %s:%d %s | %s",
			logEntry.Level, logEntry.Timestamp.AsTime().Format("2006-01-02 15:04:05"),
			logEntry.File, logEntry.Line, logEntry.Function, logEntry.Message,
		)

		log.Println("📥 Received Log:", formattedLog)

		// Send to Loki or Quickwit
		if err := sendLog(logEntry); err != nil {
			log.Printf("❌ Failed to send log: %v", err)
		}
	}
	return nil
}

// sendLog forwards logs to Loki or Quickwit
func sendLog(logEntry *pb.Log) error {
	// Convert timestamp to Unix nanoseconds (string format)
	timestamp := strconv.FormatInt(logEntry.Timestamp.AsTime().UnixNano(), 10)

	// Construct a structured log message including all fields
	logMessage := fmt.Sprintf(
		"message=%q, level=%q, timestamp=%s, hostname=%q, service=%q, file=%q, line=%d, function=%q",
		logEntry.Message, logEntry.Level, timestamp, logEntry.Hostname, logEntry.Service, logEntry.File, logEntry.Line, logEntry.Function,
	)

	// Prepare JSON payload
	payload := map[string]interface{}{
		"streams": []map[string]interface{}{
			{
				"stream": map[string]string{
					"level":    logEntry.Level,
					"service":  logEntry.Service,
					"hostname": logEntry.Hostname,
				},
				"values": [][]string{
					{timestamp, logMessage},
				},
			},
		},
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %v", err)
	}

	// Determine target URL
	targetURL := lokiURL
	if !useLoki {
		targetURL = quickwitURL
	}

	// Send HTTP request
	resp, err := http.Post(targetURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send log: %v", err)
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) // ✅ Read response body correctly
		return fmt.Errorf("unexpected response code: %d, response: %s", resp.StatusCode, string(body))
	}

	log.Println("📤 Log successfully sent to", targetURL)
	return nil
}

func main() {
	nsqAddr := "127.0.0.1:4150"
	topic := "logs_protobuf"
	channel := "log_consumer"

	_, err := NewLogConsumer(nsqAddr, topic, channel)
	if err != nil {
		log.Fatalf("❌ Failed to initialize NSQ consumer: %v", err)
	}

	log.Println("🚀 NSQ Log Consumer is running...")
	select {} // Keep running
}
