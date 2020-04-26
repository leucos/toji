package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "toji",
	Short: "CLI to add your Toggl entries to your Jira tickets",
}

var (
	configFile     string
	currentProfile string
	// Version of current binary
	Version string
	// BuildDate of current binary
	BuildDate string
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", guessConfig(), "configuration file")
	rootCmd.PersistentFlags().StringVarP(&currentProfile, "profile", "p", "", "profile to use")
}

// Run the CLI
func Run() error {
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func initConfig() {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("toji")

	err := viper.ReadInConfig() // Find and read the config file
	viper.AutomaticEnv()

	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
	}
}

func checkProfile() {
	if currentProfile == "" {
		return
	}

	if !viper.IsSet("profiles." + currentProfile) {
		fmt.Fprintf(os.Stderr, "error: profile %s not found in %s\n", currentProfile, configFile)
		os.Exit(1)
	}
}

func guessConfig() string {
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		return filepath.Clean(os.Getenv("XDG_CONFIG_HOME") + "/toji/config.yml")
	}

	return filepath.Clean(os.Getenv("HOME") + "/.config/toji/config.yml")
}

// getConfig returns the selected config in respect to the selected profile
// If the value is not found in the requested profile, the value from the
// default profile will be used.
func getConfig(c string) string {
	if viper.IsSet("profiles." + currentProfile) {
		return viper.GetString("profiles." + currentProfile + "." + c)
	}

	return viper.GetString(c)
}
