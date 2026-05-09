package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultConfigFileName = "config"

var cfgFile string

// NewRootCommand builds the top-level Cobra command and wires all subcommands.
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "bostonfear",
		Short:        "BostonFear game server and clients",
		SilenceUsage: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return initializeConfig()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Path to TOML config file (default: ./config.toml)")

	rootCmd.AddCommand(NewServerCommand())
	rootCmd.AddCommand(NewDesktopCommand())
	rootCmd.AddCommand(NewWebCommand())

	return rootCmd
}

// Execute runs the root command.
func Execute() error {
	return NewRootCommand().Execute()
}

// ExecuteWithDefaultSubcommand keeps legacy entrypoints backward compatible.
func ExecuteWithDefaultSubcommand(defaultSubcommand string, args []string) error {
	rootCmd := NewRootCommand()
	if defaultSubcommand == "" {
		rootCmd.SetArgs(args)
	} else {
		rootCmd.SetArgs(append([]string{defaultSubcommand}, args...))
	}
	return rootCmd.Execute()
}

func initializeConfig() error {
	viper.SetConfigType("toml")
	viper.SetEnvPrefix("BOSTONFEAR")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(defaultConfigFileName)
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("read config: %w", err)
	}

	return nil
}
