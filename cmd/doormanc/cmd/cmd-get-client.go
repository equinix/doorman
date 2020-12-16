package cmd

import (
	"context"
	"fmt"
	"log"

	doorman "github.com/equinix/doorman/protobuf"
	"github.com/spf13/cobra"
)

// getClientCmd represents the all command
var getClientCmd = &cobra.Command{
	Use:   "get-client",
	Short: "Get client details",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := cmd.Flags().GetString("user")
		if err != nil {
			log.Fatal(err)
		}

		conn := connectGRPC(cmd.Flags().GetString("facility"))
		resp, err := conn.GetClient(context.Background(), &doorman.GetClientRequest{
			Client: client,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(`{"status":%q, "exipires_date":%d, "revocation_date":%d, "config":%q}`+"\n",
			resp.Status.String(),
			resp.ExpiresDate,
			resp.RevocationDate,
			resp.Config,
		)
	},
}

func init() {
	getClientCmd.Flags().StringP("user", "u", "", "Equinix User UUID")
	getClientCmd.MarkFlagRequired("user")
	rootCmd.AddCommand(getClientCmd)
}
