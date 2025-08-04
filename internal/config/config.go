package config

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/vivek-pk/goadblock/internal/dbconfig"
)

func InitConfig() error {
	//VIPER Priority : flags -> env -> config -> database -> default

	// Initialize database first
	db, err := dbconfig.InitDB()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close() // Temporary connection for setup

	// Set default configuration values in database if they don't exist
	defaultConfigs := map[string]string{
		"dns_port":    "53",
		"http_port":   "8080",
		"cache_size":  "10000",
		"upstream_servers": "8.8.8.8:53,1.1.1.1:53",
	}

	for key, value := range defaultConfigs {
		_, err := dbconfig.GetConfig(db, key)
		if err == sql.ErrNoRows {
			// Config doesn't exist, set default
			err = dbconfig.SetConfig(db, key, value)
			if err != nil {
				return fmt.Errorf("failed to set default config %s: %w", key, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check config %s: %w", key, err)
		}
	}

	// Flags
	pflag.Int("dns-port", 53, "Port for the DNS server")
	pflag.Int("http-port", 8080, "Port for the HTTP server")
	pflag.String("config", "", "Config file path")

	pflag.Parse()

	bindFlagsWithFormatting(pflag.CommandLine)

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

	// Set fallback defaults first
	viper.SetDefault("http.port", 8080)
	viper.SetDefault("dns.port", 53)
	viper.SetDefault("cache.size", 10000)
	viper.SetDefault("upstream.servers", "8.8.8.8:53,1.1.1.1:53")
	viper.SetDefault("config", "")

	// Load values from database and override defaults
	for key, _ := range defaultConfigs {
		value, err := dbconfig.GetConfig(db, key)
		if err == nil {
			switch key {
			case "dns_port":
				if port, err := strconv.Atoi(value); err == nil {
					viper.Set("dns.port", port)
				}
			case "http_port":
				if port, err := strconv.Atoi(value); err == nil {
					viper.Set("http.port", port)
				}
			case "cache_size":
				if size, err := strconv.Atoi(value); err == nil {
					viper.Set("cache.size", size)
				}
			case "upstream_servers":
				viper.Set("upstream.servers", value)
			}
		}
	}

	return nil
}

func bindFlagsWithFormatting(flagSet *pflag.FlagSet) {
	flagSet.VisitAll(func(flag *pflag.Flag) {
		// Convert hyphen to dot notation for viper
		name := strings.ReplaceAll(flag.Name, "-", ".")
		viper.BindPFlag(name, flag)
	})
}

func GetDnsPort() int {
	// Check if flag was explicitly set
	if viper.IsSet("dns.port") && pflag.Lookup("dns-port").Changed {
		return viper.GetInt("dns.port")
	}
	
	// Otherwise read from database
	db, err := dbconfig.InitDB()
	if err == nil {
		defer db.Close()
		if value, err := dbconfig.GetConfig(db, "dns_port"); err == nil {
			if port, err := strconv.Atoi(value); err == nil {
				return port
			}
		}
	}
	
	// Fall back to viper (defaults)
	return viper.GetInt("dns.port")
}

func GetHttpPort() int {
	// Check if flag was explicitly set
	if viper.IsSet("http.port") && pflag.Lookup("http-port").Changed {
		return viper.GetInt("http.port")
	}
	
	// Otherwise read from database
	db, err := dbconfig.InitDB()
	if err == nil {
		defer db.Close()
		if value, err := dbconfig.GetConfig(db, "http_port"); err == nil {
			if port, err := strconv.Atoi(value); err == nil {
				return port
			}
		}
	}
	
	// Fall back to viper (defaults)
	return viper.GetInt("http.port")
}

func GetConfigPath() string {
	return viper.GetString("config")
}

func GetCacheSize() int {
	// Read from database first
	db, err := dbconfig.InitDB()
	if err == nil {
		defer db.Close()
		if value, err := dbconfig.GetConfig(db, "cache_size"); err == nil {
			if size, err := strconv.Atoi(value); err == nil {
				return size
			}
		}
	}
	
	// Fall back to viper (defaults)
	return viper.GetInt("cache.size")
}

func GetUpstreamServers() []string {
	// Read from database first
	db, err := dbconfig.InitDB()
	if err == nil {
		defer db.Close()
		if value, err := dbconfig.GetConfig(db, "upstream_servers"); err == nil {
			return strings.Split(value, ",")
		}
	}
	
	// Fall back to viper (defaults)
	serversStr := viper.GetString("upstream.servers")
	return strings.Split(serversStr, ",")
}
