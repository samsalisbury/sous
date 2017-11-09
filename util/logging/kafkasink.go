package logging

import (
	"log"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// kafkaSink
type kafkaSink struct {
	id           string
	defaultTopic string
	level        Level
	formatter    logrus.Formatter
	producer     sarama.AsyncProducer
	exit         sync.WaitGroup
}

func newKafkaSink(
	id string,
	level Level,
	formatter logrus.Formatter,
	brokers []string,
	defaultTopic string,
	injectHostname bool) (*kafkaSink, error) {

	var err error
	var producer sarama.AsyncProducer
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	kafkaConfig.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	kafkaConfig.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	if producer, err = sarama.NewAsyncProducer(brokers, kafkaConfig); err != nil {
		return nil, err
	}

	sink := &kafkaSink{
		id:           id,
		defaultTopic: defaultTopic,
		level:        level,
		formatter:    formatter,
		producer:     producer,
	}

	sink.exit.Add(1)

	go func() {
		defer sink.exit.Done()
		for err := range producer.Errors() {
			log.Printf("Failed to send log entry to Kafka: %v\n", err)
		}
	}()

	return sink, nil
}

func (sink *kafkaSink) ID() string {
	return sink.id
}

func (sink kafkaSink) shouldSend(lvl Level) bool {
	return lvl <= sink.level
}

func (sink *kafkaSink) send(lvl Level, entry *logrus.Entry) error {
	var partitionKey sarama.ByteEncoder
	var b []byte
	var err error

	if !sink.shouldSend(lvl) {
		return nil
	}

	uuid, present := entry.Data["@uuid"]
	if !present {
		return errors.Errorf("Required field @uuid absent")
	}
	uuidStr, stringType := uuid.(string)
	if !stringType {
		return errors.Errorf("Required field @uuid was type %T, not string", uuid)
	}
	partitionKey = sarama.ByteEncoder(uuidStr)

	entry.Level = lvl.logrusLevel()
	if b, err = sink.formatter.Format(entry); err != nil {
		return err
	}
	value := sarama.ByteEncoder(b)

	topic := sink.defaultTopic
	sink.producer.Input() <- &sarama.ProducerMessage{
		Key:   partitionKey,
		Topic: topic,
		Value: value,
	}
	return nil
}

func (sink *kafkaSink) closedown() {
	sink.producer.AsyncClose()
	sink.exit.Wait()
}
