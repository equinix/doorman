package cmd

import (
	"context"
	"log"
	"net"
	"os"

	doorman "github.com/equinix/doorman/protobuf"
	"github.com/spf13/cobra"
)

// authCmd represents the all command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate client connection",
	Run: func(cmd *cobra.Command, args []string) {
		conn := connectGRPC(cmd.Flags().GetString("facility"))

		file, err := cmd.Flags().GetString("creds")
		if err != nil {
			log.Fatal(err)
		}

		client, err := cmd.Flags().GetString("user")
		if err != nil {
			log.Fatal(err)
		}

		ip, err := cmd.Flags().GetIP("ip")
		if err != nil {
			log.Fatal(err)
		}

		resp, err := conn.Authenticate(context.Background(), &doorman.AuthenticateRequest{
			File:         file,
			Client:       client,
			ConnectingIp: ip.String(),
		})
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(int(resp.Status))
	},
}

func init() {
	authCmd.Flags().StringP("creds", "c", "", "client credentials file")
	authCmd.Flags().IPP("ip", "i", net.IPv4zero, "client source ip")
	authCmd.Flags().StringP("user", "u", "", "client user id")
	authCmd.MarkFlagRequired("creds")
	authCmd.MarkFlagRequired("ip")
	authCmd.MarkFlagRequired("user")
	rootCmd.AddCommand(authCmd)
}
