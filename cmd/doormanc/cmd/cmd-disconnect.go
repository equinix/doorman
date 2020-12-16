package cmd

import (
	"context"
	"log"
	"os"

	doorman "github.com/equinix/doorman/protobuf"
	"github.com/spf13/cobra"
)

// disconnectCmd represents the all command
var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect connection",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := cmd.Flags().GetString("user")
		if err != nil {
			log.Fatal(err)
		}

		conn := connectGRPC(cmd.Flags().GetString("facility"))
		resp, err := conn.Disconnect(context.Background(), &doorman.DisconnectRequest{
			Client: client,
		})
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(int(resp.Status))
	},
}

func init() {
	disconnectCmd.Flags().StringP("user", "u", "", "client user id")
	disconnectCmd.MarkFlagRequired("user")
	rootCmd.AddCommand(disconnectCmd)
}
