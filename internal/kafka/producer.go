package kafka

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "sync"
    "time"

    "github.com/segmentio/kafka-go"
)

var (
    writers   = map[string]*kafka.Writer{}
    writersMu sync.RWMutex
    brokers   []string
    enabled   bool
)

// Init reads KAFKA_BROKERS env var (comma-separated) and marks Kafka enabled.
// If the env var is not set, all Publish calls become silent no-ops.
func Init() {
	addr := os.Getenv("KAFKA_BROKERS")
	if addr == "" {
		log.Println("KAFKA_BROKERS not set — Kafka disabled")
        return
	}
	// split by comma
	for _, b := range splitCSV(addr) {
		brokers = append(brokers, b)
	}
	enabled = true
	log.Printf("✅ Kafka enabled, brokers: %v", brokers)
}

func splitCSV(s string) []string {
    var out []string
    cur := ""
    for _, c := range s {
        if c == ',' {
            if cur != "" {
                out = append(out, cur)
                cur = ""
            }
        } else {
            cur += string(c)
        }
    }
    if cur != "" {
        out = append(out, cur)
    }
    return out
}

func writerFor(topic string) *kafka.Writer {
    writersMu.RLock()
    w, ok := writers[topic]
    writersMu.RUnlock()
    if ok {
        return w
    }
    writersMu.Lock()
    defer writersMu.Unlock()
	if w, ok := writers[topic]; ok {
		return w
	}
    w = &kafka.Writer{
        Addr:         kafka.TCP(brokers...),
        Topic:        topic,
        Balancer:     &kafka.LeastBytes{},
        WriteTimeout: 5 * time.Second,
        ReadTimeout:  5 * time.Second,
    }
    writers[topic] = w
    return w
}

// Publish serialises payload as JSON and writes it to the topic.
// Silently no-ops when Kafka is disabled.
func Publish(ctx context.Context, topic string, payload any) {
	if !enabled {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("kafka.Publish marshal error: %v", err)
        return
	}
	w := writerFor(topic)
	if err := w.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		log.Printf("kafka.Publish [%s] error: %v", topic, err)
	}
}

func Close() {
	writersMu.Lock()
	defer writersMu.Unlock()
	for topic, w := range writers {
		if err := w.Close(); err != nil {
			log.Printf("kafka: error closing writer for %s: %v", topic, err)
		}
		delete(writers, topic)
	}
}
