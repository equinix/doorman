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

type sortableAllocs []*doorman.Allocation

func (c sortableAllocs) Len() int {
	return len([]*doorman.Allocation(c))
}
func (c sortableAllocs) Less(i, j int) bool {
	switch bytes.Compare(net.ParseIP(c[i].Ip), net.ParseIP(c[j].Ip)) {
	case -1:
		return true
	case 1:
		return false
	default:
		return c[i].Client < c[j].Client
	}
}
func (c sortableAllocs) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// listAllocsCmd represents the all command
var listAllocsCmd = &cobra.Command{
	Use:   "list-allocs",
	Short: "List ip pool and possibly allocated clients (sorted by ip address)",
	Run: func(cmd *cobra.Command, args []string) {
		conn := connectGRPC(cmd.Flags().GetString("facility"))
		resp, err := conn.ListAllocations(context.Background(), &doorman.ListAllocationsRequest{})
		if err != nil {
			log.Fatal(err)
		}

		sort.Sort(sortableAllocs(resp.Allocations))
		for _, alloc := range resp.Allocations {
			fmt.Printf(`{"ip":"%s", "client":"%s"}`+"\n", alloc.Ip, alloc.Client)
		}
	},
}

func init() {
	rootCmd.AddCommand(listAllocsCmd)
}
