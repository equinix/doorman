package cmd

import (
	"fmt"
	"log"
	"os"

	doormanc "github.com/equinix/doorman/client"
	doorman "github.com/equinix/doorman/protobuf"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "doormanc",
	Short: "A hopefully easy to use cli for doorman interaction",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func connectGRPC(facility string, err error) doorman.VPNServiceClient {
	if err != nil {
		panic(err)
	}

	if os.Getenv("GRPC_INSECURE") == "" {
		if cert, ok := certs[facility]; ok {
			os.Setenv("GRPC_CERT", cert)
		}
	}

	client, err := doormanc.New(facility)
	if err != nil {
		log.Fatal(errors.Wrap(err, "connect to doorman server"))
	}
	return client
}

func init() {
	log.SetFlags(0)
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "facility", "f", "", "used to build grcp and http urls")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
