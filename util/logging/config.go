package logging

// Config captures outside configuration for a root LogSet
type Config struct {
	Kafka struct {
		DefaultLevel string `env:"SOUS_KAFKA_LOG_LEVEL"`
		Topic        string `env:"SOUS_KAFKA_TOPIC"`
		Brokers      []string
		BrokerList   string `env:"SOUS_KAFKA_BROKERS"`
	}
}

func (cfg *Config) Fixup() {
	if len(cfg.Brokers) == 0 && cfg.BrokerList != "" {
		cfg.Brokers = strings.Split(cfg.BrokerList, ",")
	}
}
