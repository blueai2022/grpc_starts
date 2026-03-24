package config

import "time"

type Settings struct {
	HTTP HTTPSettings `json:"http" yaml:"http"`
}

type HTTPSettings struct {
	Host            string        `json:"host" yaml:"host"`
	Port            int           `json:"port" yaml:"port"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
}
