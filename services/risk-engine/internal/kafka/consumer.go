package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"
	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/processor"
)

type Consumer struct {
	reader         *kafkago.Reader
	processor      processor.EventProcessor
	processTimeout time.Duration
	workerBuffer   int

	mu        sync.Mutex
	workers   map[int]chan kafkago.Message
	workersWg sync.WaitGroup
	commitMu  sync.Mutex
}

func NewConsumer(brokers []string, topic, groupID string, workerBuffer int, processTimeout time.Duration, eventProcessor processor.EventProcessor) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:                brokers,
		Topic:                  topic,
		GroupID:                groupID,
		MinBytes:               1,
		MaxBytes:               10e6,
		MaxWait:                500 * time.Millisecond,
		ReadBackoffMin:         200 * time.Millisecond,
		ReadBackoffMax:         1 * time.Second,
		GroupBalancers:         []kafkago.GroupBalancer{kafkago.RangeGroupBalancer{}},
		WatchPartitionChanges:  true,
		PartitionWatchInterval: 5 * time.Second,
		StartOffset:            kafkago.FirstOffset,
	})

	return &Consumer{
		reader:         reader,
		processor:      eventProcessor,
		processTimeout: processTimeout,
		workerBuffer:   workerBuffer,
		workers:        make(map[int]chan kafkago.Message),
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	log.Printf("consumer started with partition workers and consumer group")

	for {
		select {
		case <-ctx.Done():
			c.stopWorkers()
			return nil
		case err := <-errCh:
			c.stopWorkers()
			return err
		default:
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				c.stopWorkers()
				return nil
			}
			c.stopWorkers()
			return fmt.Errorf("fetch message: %w", err)
		}

		worker := c.getOrCreatePartitionWorker(msg.Partition, errCh)
		select {
		case worker <- msg:
		case <-ctx.Done():
			c.stopWorkers()
			return nil
		case err := <-errCh:
			c.stopWorkers()
			return err
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) getOrCreatePartitionWorker(partition int, errCh chan<- error) chan kafkago.Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	if worker, ok := c.workers[partition]; ok {
		return worker
	}

	worker := make(chan kafkago.Message, c.workerBuffer)
	c.workers[partition] = worker
	c.workersWg.Add(1)

	go func(partition int, messages <-chan kafkago.Message) {
		defer c.workersWg.Done()
		log.Printf("partition worker started partition=%d", partition)
		for msg := range messages {
			if err := c.processAndCommit(msg); err != nil {
				nonBlockingSend(errCh, fmt.Errorf("partition=%d offset=%d process failed: %w", partition, msg.Offset, err))
				return
			}
		}
		log.Printf("partition worker stopped partition=%d", partition)
	}(partition, worker)

	return worker
}

func (c *Consumer) processAndCommit(msg kafkago.Message) error {
	var event model.Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("decode event JSON: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.processTimeout)
	defer cancel()

	if err := c.processor.Process(ctx, event, msg.Partition, msg.Offset); err != nil {
		return fmt.Errorf("process event event_id=%s: %w", event.EventID, err)
	}

	c.commitMu.Lock()
	defer c.commitMu.Unlock()
	if err := c.reader.CommitMessages(context.Background(), msg); err != nil {
		return fmt.Errorf("commit message event_id=%s: %w", event.EventID, err)
	}

	return nil
}

func (c *Consumer) stopWorkers() {
	c.mu.Lock()
	for partition, worker := range c.workers {
		close(worker)
		delete(c.workers, partition)
	}
	c.mu.Unlock()
	c.workersWg.Wait()
}

func nonBlockingSend(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}
