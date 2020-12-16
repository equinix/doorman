package cmd

import (
	"context"
	"fmt"
	"log"

	doorman "github.com/equinix/doorman/protobuf"
	"github.com/spf13/cobra"
)

// createClientCmd represents the all command
var createClientCmd = &cobra.Command{
	Use:   "create-client",
	Short: "Create client configuration",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := cmd.Flags().GetString("user")
		if err != nil {
			log.Fatal(err)
		}

		conn := connectGRPC(cmd.Flags().GetString("facility"))
		resp, err := conn.CreateClient(context.Background(), &doorman.CreateClientRequest{
			Client: client,
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(resp.Config)
	},
}

func init() {
	createClientCmd.Flags().StringP("user", "u", "", "Equinix User UUID")
	createClientCmd.MarkFlagRequired("user")
	rootCmd.AddCommand(createClientCmd)
}
