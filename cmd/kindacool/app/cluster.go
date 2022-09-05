package app

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/brumhard/kindacool/pkg/kindacool"

	"github.com/spf13/cobra"
)

const outputPrefixChangeInterval = 100 * time.Millisecond

func BuildClusterCommand() *cobra.Command {
	manager := &kindacool.Manager{}

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: fmt.Sprintf("%s cluster is the main entrypoint to all cluster management operations", CLI),
		Long: `The cluster subcommand includes all the cluster management operations.

By default the name "kindacool" is used for all cluster operations.
If you want to manage a second cluster set the --name flag.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Enable swapping out stdout/stderr for testing
			manager.Logger = log.New(cmd.OutOrStderr(), "🚀 ", 0)
			// use random emoji as prefix
			go func(ctx context.Context) {
				emojis := []string{"🚀", "💃", "✨", "🔥", "🦥", "👽", "👾", "👀", "💅"}
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(outputPrefixChangeInterval):
					}
					//nolint:gosec // not relevant for security
					manager.Logger.SetPrefix(emojis[rand.Intn(len(emojis)-1)] + " ")
				}
			}(cmd.Context())

			if err := kindacool.EnsureEnvironment(cmd.Context()); err != nil {
				return err
			}

			return manager.Options.Validate()
		},
	}

	cmd.PersistentFlags().StringVarP(&manager.Options.Name, "name", "n", "kindacool", "Name of the cluster to manage.")
	cmd.PersistentFlags().BoolVarP(&manager.Options.Verbose, "verbose", "v", false, "Enable verbose pulumi output.")

	cmd.AddCommand(BuildCreateCommand(manager))
	cmd.AddCommand(BuildDestroyCommand(manager))
	cmd.AddCommand(BuildLsCommand(manager))
	cmd.AddCommand(BuildKubeconfigCommand(manager))
	cmd.AddCommand(BuildSSHKeyCommand(manager))

	return cmd
}
