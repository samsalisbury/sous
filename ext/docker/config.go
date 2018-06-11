package docker

type Config struct {
	RegistryHost string `env:"SOUS_DOCKER_REGISTRY_HOST"`
}

// DefaultConfig builds a default configuration, which can be then overridden by
// client code.
func DefaultConfig() Config {
	return Config{
		RegistryHost: "docker.otenv.com",
	}
}
