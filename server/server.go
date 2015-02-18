package server

type Server interface {
	Initialise(*Config)
	Run()

	Name() string
	Description() string

	RegisterEndpoint(endpoint Endpoint)
	DeregisterEndpoint(pattern string)
}

// DefaultServer stores a default implementation, for simple usage
var DefaultServer Server = NewAMQPServer()

// Initialise our DefaultServer with a Config
func Initialise(c *Config) {
	DefaultServer.Initialise(c)
}

// RegisterEndpoint with the DefaultServer
func RegisterEndpoint(endpoint Endpoint) {
	DefaultServer.RegisterEndpoint(endpoint)
}

// Run the DefaultServer
func Run() {
	DefaultServer.Run()
}

// Config defines the config a server needs to start up, and serve requests
type Config struct {
	Name        string
	Description string
}
