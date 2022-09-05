package app

import (
	"fmt"
	"strings"

	"github.com/brumhard/kindacool/pkg/kindacool"

	"github.com/spf13/cobra"
)

func BuildLsCommand(manager *kindacool.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List all k3s clusters on OpenStack",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			clusters, err := manager.List(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), strings.Join(clusters, "\n"))
			return nil
		},
	}

	return cmd
}
