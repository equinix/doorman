package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"sort"

	doorman "github.com/equinix/doorman/protobuf"
	"github.com/spf13/cobra"
)

type sortableConnections []*doorman.Connection

func (c sortableConnections) Len() int {
	return len([]*doorman.Connection(c))
}
func (c sortableConnections) Less(i, j int) bool {
	switch bytes.Compare(net.ParseIP(c[i].Allocation.Ip), net.ParseIP(c[j].Allocation.Ip)) {
	case -1:
		return true
	case 1:
		return false
	}
	if c[i].Client < c[j].Client {
		return true
	}
	if c[j].Client < c[i].Client {
		return false
	}
	return c[i].Since < c[j].Since
}
func (c sortableConnections) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// listConnectionsCmd represents the all command
var listConnectionsCmd = &cobra.Command{
	Use:   "list-connections",
	Short: "List active vpn connections (sorted by ip address, then client-id, finally by connection time)",
	Run: func(cmd *cobra.Command, args []string) {
		conn := connectGRPC(cmd.Flags().GetString("facility"))
		resp, err := conn.ListConnections(context.Background(), &doorman.ListConnectionsRequest{})
		if err != nil {
			log.Fatal(err)
		}

		sort.Sort(sortableConnections(resp.Connections))
		for _, conn := range resp.Connections {
			fmt.Printf(`{"id":%q, "allocation":%q, "source":%q, "since":%d}`+"\n",
				conn.Client,
				conn.Allocation.Ip,
				conn.ConnectingIp,
				conn.Since,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(listConnectionsCmd)
}
