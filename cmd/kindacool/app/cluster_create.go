package app

import (
	"os"

	"github.com/brumhard/kindacool/pkg/k3s"
	"github.com/brumhard/kindacool/pkg/kindacool"

	"github.com/spf13/cobra"
)

func BuildCreateCommand(manager *kindacool.Manager) *cobra.Command {
	clusterArgs := &k3s.ClusterArgs{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a k3s cluster on OpenStack",
		Long: `The create command creates a new k3s single node cluster on OpenStack.

It will first create all the required resources like a VM and security groups on OpenStack
and then install k3s on top of it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager.Run(cmd.Context(), clusterArgs); err != nil {
				return err
			}

			kubeconfig, err := manager.FetchOutput(cmd.Context(), kindacool.OutputKubeconfig)
			if err != nil {
				return err
			}

			kubeconfigFile, err := KubeconfigFile(manager.Options.Name)
			if err != nil {
				return err
			}

			cmd.Printf("Writing kubeconfig to %q\n", kubeconfigFile)
			//nolint:gomnd // well-known file permissions
			return os.WriteFile(kubeconfigFile, []byte(kubeconfig), 0600)
		},
	}

	cmd.Flags().IntSliceVarP(
		&clusterArgs.AdditionalPorts,
		"additionalPorts", "p", nil,
		`By default only the ports 22, 80, 443 and 6443 are open in the security group.
To open additional ports for inbound traffic define them here.
The flag can be defined multiple times like -p 1234 -p 2345`,
	)

	cmd.Flags().StringVarP(
		&clusterArgs.MachineFlavor,
		"flavor", "f", "m4.large",
		"OpenStack flavor to be used for the machines. Use 'openstack flavor list' to obtain a list of all flavors.",
	)

	cmd.Flags().BoolVar(
		&clusterArgs.Public,
		"public", false,
		"Make the cluster reachable from the internet with a floating IP.",
	)

	cmd.Flags().IntVarP(
		&clusterArgs.NodeCount,
		"nodeCount", "c", 1,
		`Amount of nodes to create and join to a cluster.
If the count is >1 additional worker nodes will be joined to a single master node.`,
	)

	cmd.Flags().IntVar(
		&clusterArgs.VolumeSize,
		"volumeSize", 0,
		`Size in GigaBytes (GB) that will be added to the boot volume.
If the size is <=0 no additional volume will be created.`,
	)

	cmd.Flags().StringVar(
		&clusterArgs.MachineImage,
		"machineImage", "Ubuntu 22.04",
		"Openstack image that will be used for the nodes. Use 'openstack image list' to obtain a list of all images.",
	)

	cmd.Flags().StringVar(
		&clusterArgs.MachineUser,
		"machineUser", "ubuntu",
		"User that sets up k3s via SSH.",
	)

	cmd.Flags().StringVar(
		&clusterArgs.PrivateNetworkName,
		"privateNetworkName", "",
		"Private network to use when not exposing to public.",
	)

	cmd.Flags().StringVar(
		&clusterArgs.PublicIPPool,
		"publicIPPool", "",
		"Public IP pool to use when exposing to public.",
	)

	cmd.Flags().StringVar(
		&clusterArgs.PublicNetworkName,
		"publicNetworkName", "",
		"Network name that is exposed to the internet.",
	)

	cmd.Flags().StringVar(
		&clusterArgs.PublicNetworkID,
		"publicNetworkID", "",
		"Network ID that is exposed to the internet.",
	)

	return cmd
}
