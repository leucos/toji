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
	// currentConfiguration *Configuration
)

// // Configuration selected
// type Configuration struct {
// 	Jira  JiraConfig  `yaml:"jira"`
// 	Toggl TogglConfig `yaml:"toggl"`
// }

// // JiraConfig holds a Jira configuration
// type JiraConfig struct {
// 	URL   string `yaml:"url"`
// 	Token string `yaml:"token"`
// }

// // TogglConfig holds a Toggl configuration
// type TogglConfig struct {
// 	Token string `yaml:"token"`
// }

func init() {
	cobra.OnInitialize(initConfig, checkProfile)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", guessConfig(), "configuration file")
	rootCmd.PersistentFlags().StringVar(&currentProfile, "profile", "", "profile to use")
}

// Run the CLI
func Run() error {
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(syncCmd)

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
func getConfig(c string) string {
	if currentProfile == "" {
		return viper.GetString(c)
	}

	if viper.IsSet("profiles." + currentProfile) {
		return viper.GetString("profiles." + currentProfile + "." + c)
	}
	return ""
}
