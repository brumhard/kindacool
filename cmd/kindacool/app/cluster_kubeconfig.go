package app

import (
	"fmt"

	"github.com/brumhard/kindacool/pkg/kindacool"

	"github.com/spf13/cobra"
)

func BuildKubeconfigCommand(manager *kindacool.Manager) *cobra.Command {
	var export bool

	cmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "Output a cluster's kubeconfig",
		Long: fmt.Sprintf(`The kubeconfig command has to modes of working.

1. By default the command will fetch the cluster's kubeconfig from Pulumi and write it to stdout.
   Generally, the kubeconfig will also be written to ~/.kube/%[1]s-<clustername>.yaml after creation.

2. If the --export flag is set it will instead output a shell command to set $KUBECONFIG to the cluster's kubeconfig's default file path.
   This can be used like:
   	$ eval "$(%[1]s cluster kubeconfig --export)"
`, CLI),

		RunE: func(cmd *cobra.Command, args []string) error {
			if export {
				kubeconfigFile, err := KubeconfigFile(manager.Options.Name)
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), `export KUBECONFIG="%s"`, kubeconfigFile)
				return nil
			}

			kubeconfig, err := manager.FetchOutput(cmd.Context(), kindacool.OutputKubeconfig)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), kubeconfig)
			return nil
		},
	}

	cmd.Flags().BoolVar(&export, "export", false, "Output an export string pointing to the file instead of the kubeconfig itself.")

	return cmd
}
