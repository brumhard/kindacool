package app

import (
	"os"

	"github.com/brumhard/kindacool/pkg/kindacool"

	"github.com/spf13/cobra"
)

func BuildDestroyCommand(manager *kindacool.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "destroy",
		Aliases: []string{"rm"},
		Short:   "Destroys a k3s cluster on OpenStack",
		Long:    "Destroys a k3s cluster on OpenStack",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager.Destroy(cmd.Context()); err != nil {
				return err
			}

			kubeconfigFile, err := KubeconfigFile(manager.Options.Name)
			if err != nil {
				return err
			}

			// ignore errors since they will only occur if the file is not there
			_ = os.Remove(kubeconfigFile)

			return nil
		},
	}

	return cmd
}
