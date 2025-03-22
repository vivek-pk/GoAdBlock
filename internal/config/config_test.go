package config

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func resetViper() {
	viper.Reset()
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	os.Args = []string{"cmd"}
}

func TestConfigDefaults(t *testing.T) {
	resetViper()

	err := InitConfig()
	assert.NoError(t, err)
	assert.EqualValues(t, 53, GetDnsPort())
	assert.EqualValues(t, 8080, GetHttpPort())
}

func TestConfigFromFlags(t *testing.T) {
	resetViper()

	os.Args = []string{"cmd", "--dns-port=54", "--http-port=3000"}

	err := InitConfig()
	assert.NoError(t, err)
	assert.EqualValues(t, 54, GetDnsPort())
	assert.EqualValues(t, 3000, GetHttpPort())
}

func TestConfigFromEnvVars(t *testing.T) {
	resetViper()

	os.Setenv("GOADBLOCK_DNS_PORT", "55")
	os.Setenv("GOADBLOCK_HTTP_PORT", "3001")

	err := InitConfig()
	assert.NoError(t, err)
	assert.EqualValues(t, 55, GetDnsPort())
	assert.EqualValues(t, 3001, GetHttpPort())

	os.Unsetenv("GOADBLOCK_DNS_PORT")
	os.Unsetenv("GOADBLOCK_HTTP_PORT")
}

func TestConfigFromConfigFile(t *testing.T) {
	resetViper()
	configContent := `dns:
  port: 50
  upstream: '8.8.8.8'
  cache_size: 5000
  cache_ttl: 3600

http:
  port: 5000 
  username: 'admin'
  password: 'changeme'`

	tmpfile, err := os.CreateTemp("", "config*.yaml")
	assert.NoError(t, err, "Should create temp file")
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(configContent)
	assert.NoError(t, err, "Should write to temp file")
	tmpfile.Close()

	// provide temp file as env
	os.Setenv("GOADBLOCK_CONFIG", tmpfile.Name())

	err = InitConfig()
	assert.NoError(t, err)
	assert.EqualValues(t, 50, GetDnsPort())
	assert.EqualValues(t, 5000, GetHttpPort())

	os.Unsetenv("GOADBLOCK_CONFIG")
}

func TestPriorityOrder(t *testing.T) {
	resetViper()

	// Flags
	os.Args = []string{"cmd", "--dns-port=60"}

	// Env Vars
	os.Setenv("GOADBLOCK_DNS_PORT", "55")
	os.Setenv("GOADBLOCK_HTTP_PORT", "3678")

	//Config File
	configContent := `dns:
  port: 50
  upstream: '8.8.8.8'
  cache_size: 5000
  cache_ttl: 3600

http:
  port: 5000 
  username: 'admin'
  password: 'changeme'`
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	assert.NoError(t, err, "Should create temp file")
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(configContent)
	assert.NoError(t, err, "Should write to temp file")
	tmpfile.Close()

	// provide temp file as env
	os.Setenv("GOADBLOCK_CONFIG", tmpfile.Name())

	err = InitConfig()
	assert.NoError(t, err)
	assert.EqualValues(t, 60, GetDnsPort())
	assert.EqualValues(t, 3678, GetHttpPort())

	os.Unsetenv("GOADBLOCK_DNS_PORT")
	os.Unsetenv("GOADBLOCK_HTTP_PORT")
	os.Setenv("GOADBLOCK_CONFIG", tmpfile.Name())
}
