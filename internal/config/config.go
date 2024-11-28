package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	file    string
	profile string
}

var Current Config

func Init(configFile, profile string) {
	Current.file = configFile
	Current.profile = profile

	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("toji")

	err := viper.ReadInConfig() // Find and read the config file
	viper.AutomaticEnv()

	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		fmt.Fprintf(os.Stderr, "error: configuration %s file not found: %v\n", configFile, err)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to parse configuration %s file: %v\n", configFile, err)
		os.Exit(1)
	}
}

func (c Config) CheckProfile() {
	if c.profile == "" {
		return
	}

	if !viper.IsSet("profiles." + c.profile) {
		fmt.Fprintf(os.Stderr, "error: profile %s not found in %s\n", c.profile, c.file)
		os.Exit(1)
	}
}

func (c Config) Guess() string {
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		return filepath.Clean(os.Getenv("XDG_CONFIG_HOME") + "/toji/config.yml")
	}

	return filepath.Clean(os.Getenv("HOME") + "/.config/toji/config.yml")
}

// Get returns the selected config in respect to the selected profile
// If the value is not found in the requested profile, the value from the
// default profile will be used.
func (c Config) Get(key string) string {
	if viper.IsSet("profiles." + c.profile) {
		return viper.GetString("profiles." + c.profile + "." + key)
	}

	return viper.GetString(key)
}

// checkConfig checks if a configuration value is set
func (c Config) Check(key string) bool {
	if viper.IsSet("profiles." + c.profile) {
		return viper.IsSet("profiles." + c.profile + "." + key)
	}

	return viper.IsSet(key)
}
