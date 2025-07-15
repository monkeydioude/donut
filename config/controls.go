package config

var Controls *ControlsConfig = &ControlsConfig{
	KeyMapping: make(map[string][]string),
}

type ControlsConfig struct {
	KeyMapping map[string][]string `yaml:"KeyMapping"`
}
