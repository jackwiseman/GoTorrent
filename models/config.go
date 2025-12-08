package models

type Config struct {
	Connections int // Number of connections to use
}

func NewConfig(connections int) *Config {
	return &Config{
		Connections: connections,
	}
}
