package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

const CLI = "kindacool"

func BuildRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   CLI,
		Short: fmt.Sprintf("%s can be used to quickly setup new Kubernetes (k3s) clusters on OpenStack.", CLI),
		Long: fmt.Sprintf(`%[1]s is a CLI to quickly setup new Kubernetes (k3s) clusters on OpenStack.
It uses Pulumi Automation API in the background.
Before usage it is required to install pulumi and login to a backend. Also you need to source an .openrc file.

A cluster can then be setup with:
	$ %[1]s cluster create

After the cluster is created set the KUBECONFIG env var with:
	$ eval "$(%[1]s cluster kubeconfig --export)"

For more information, please visit the project's homepage: https://github.com/brumhard/kindacool.
		`, CLI),
		// don't show errors and usage on errors in any RunE function.
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(BuildVersionCommand())
	cmd.AddCommand(BuildClusterCommand())

	return cmd
}
