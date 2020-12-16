package cmd

import (
	"context"
	"log"
	"os"

	doorman "github.com/equinix/doorman/protobuf"
	"github.com/spf13/cobra"
)

// revokeClientCmd represents the all command
var revokeClientCmd = &cobra.Command{
	Use:   "revoke-client",
	Short: "Revoke client config, including certificates",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := cmd.Flags().GetString("user")
		if err != nil {
			log.Fatal(err)
		}
		conn := connectGRPC(cmd.Flags().GetString("facility"))
		resp, err := conn.RevokeClient(context.Background(), &doorman.RevokeClientRequest{
			Client: client,
		})
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(int(resp.Status))
	},
}

func init() {
	revokeClientCmd.Flags().StringP("user", "u", "", "client user id")
	revokeClientCmd.MarkFlagRequired("user")
	rootCmd.AddCommand(revokeClientCmd)
}
