package kafka

import (
    "context"
    "encoding/json"
    "log"
    "time"

    "github.com/segmentio/kafka-go"
)

// HandlerFunc is called for each message on a topic.
type HandlerFunc func(ctx context.Context, msg []byte) error

// StartConsumer launches a long-running reader goroutine for the given topic+groupID.
// It retries on transient errors with a short backoff.
// Pass the root context from main(); the goroutine exits when ctx is cancelled.
func StartConsumer(ctx context.Context, topic, groupID string, handler HandlerFunc) {
	if !enabled {
		return
	}
	go func () {
		r := kafka.NewReader(kafka.ReaderConfig{
            Brokers:        brokers,
            Topic:          topic,
            GroupID:        groupID,
            MinBytes:       1,
            MaxBytes:       1e6, // 1 MB
            CommitInterval: time.Second,
        })
		defer r.Close()
		log.Printf("kafka consumer [%s/%s] started", topic, groupID)
		for {
			m, err := r.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return //graceful shutdown
				}
				log.Printf("kafka consumer [%s] read error: %v — retrying in 2s", topic, err)
                time.Sleep(2 * time.Second)
                continue
			}
			if err := handler(ctx, m.Value); err != nil {
				log.Printf("kafka consumer [%s] handler error: %v", topic, err)
			}
		}
	}()
}

// DecodeJSON is a helper to unmarshal a raw message into a typed struct.
func DecodeJSON(raw []byte, out any) error {
    return json.Unmarshal(raw, out)
}