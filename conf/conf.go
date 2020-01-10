package conf

var Conf = &Config{}

type Config struct {
	Name       string
	Version    string
	Kubeconfig string
}