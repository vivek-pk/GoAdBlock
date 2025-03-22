package config

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func InitConfig() error {
	//VIPER Priority : flags -> env -> config -> default

	// Flags
	pflag.Int("dns-port", 53, "Port for the DNS server")
	pflag.Int("http-port", 8080, "Port for the HTTP server")
	pflag.String("config", "", "Config file path")

	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return fmt.Errorf("error binding pflags: %w", err)
	}

	// Env
	viper.SetEnvPrefix("GOADBLOCK")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// Config
	configPath := viper.GetString("config")
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.goablock")
		viper.AddConfigPath("/etc/goablock")
	}

	if err := viper.ReadInConfig(); err != nil {
		// It's okay if the config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Default Values
	viper.SetDefault("http-port", 8080)
	viper.SetDefault("dns-port", 5345)
	viper.SetDefault("config", "")

	return nil
}

func GetDnsPort() int {
	return viper.GetInt("dns-port")
}

func GetHttpPort() int {
	return viper.GetInt("http-port")
}

func GetConfigPath() string {
	return viper.GetString("config")
}
