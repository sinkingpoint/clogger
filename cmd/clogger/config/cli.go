package config

var CLI struct {
	Server struct {
		MetricsAddress string `help:"The Address to serve Prometheus Metrics on" default:":4280"`
	} `cmd:"" help:"Start the Logging Server" default:"1"`
}
