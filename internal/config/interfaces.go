package config

// Loader abstracts configuration loading for testability.
type Loader interface {
	Load(dir string) (*Config, error)
}
