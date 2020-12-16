package cmd

import (
	"context"
	"fmt"
	"log"
	"sort"

	doorman "github.com/equinix/doorman/protobuf"
	"github.com/spf13/cobra"
)

type sortableClients []*doorman.Client

func (c sortableClients) Len() int {
	return len([]*doorman.Client(c))
}
func (c sortableClients) Less(i, j int) bool {
	if c[i].Client < c[j].Client {
		return true
	}
	if c[j].Client < c[i].Client {
		return false
	}
	if c[i].Status < c[j].Status {
		return true
	}
	if c[j].Status < c[i].Status {
		return false
	}
	return c[i].ExpiresDate < c[j].ExpiresDate
}
func (c sortableClients) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// listClientsCmd represents the all command
var listClientsCmd = &cobra.Command{
	Use:   "list-clients",
	Short: "List known clients (sorted by client-id then status)",
	Run: func(cmd *cobra.Command, args []string) {
		conn := connectGRPC(cmd.Flags().GetString("facility"))
		resp, err := conn.ListClients(context.Background(), &doorman.ListClientsRequest{})
		if err != nil {
			log.Fatal(err)
		}

		sort.Sort(sortableClients(resp.Clients))

		for _, client := range resp.Clients {
			fmt.Printf(`{"id":%q, "status":%q, "expires_date":%d, "revocation_date":%d}`+"\n",
				client.Client,
				client.Status.String(),
				client.ExpiresDate,
				client.RevocationDate,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(listClientsCmd)
}
