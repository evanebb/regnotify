package configuration

import "github.com/spf13/viper"

type Configuration struct {
	Log     Log
	HTTP    HTTP
	Storage Storage
}

// SetDefaults sets the defaults for the configuration on a viper.Viper instance.
func SetDefaults(v *viper.Viper) {
	v.SetDefault("log.level", "info")
	v.SetDefault("log.formatter", "text")
	v.SetDefault("http.addr", ":8000")
	v.SetDefault("storage.bolt.path", "/var/lib/regnotify/events.db")
}

type Log struct {
	Level     string
	Formatter string
}

type HTTP struct {
	Addr        string
	Certificate string
	Key         string
}

type Storage struct {
	Bolt BoltDBStorage
}

type BoltDBStorage struct {
	Enabled bool
	Path    string
}
