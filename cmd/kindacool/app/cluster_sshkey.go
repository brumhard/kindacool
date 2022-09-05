package app

import (
	"fmt"

	"github.com/brumhard/kindacool/pkg/kindacool"

	"github.com/spf13/cobra"
)

func BuildSSHKeyCommand(manager *kindacool.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sshkey",
		Short: "Output a cluster's ssh-key",
		Long:  "The command will fetch the cluster's ssh-key from Pulumi and write it to stdout.",

		RunE: func(cmd *cobra.Command, args []string) error {
			sshkey, err := manager.FetchOutput(cmd.Context(), kindacool.OutputSSHKey)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), sshkey)
			return nil
		},
	}

	return cmd
}
