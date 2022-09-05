package k3s

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-openstack/sdk/v3/go/openstack/compute"
	"github.com/pulumi/pulumi-openstack/sdk/v3/go/openstack/images"
	"github.com/pulumi/pulumi-openstack/sdk/v3/go/openstack/networking"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	sshPort = 22
)

var ErrInvalidNodeCount = errors.New("invalid amount of nodes, must be > 0")
var ErrImageNotFound = errors.New("image not found by name")
var ErrPrivateNetwork = errors.New("privateNetworkName is empty (required if public is set to false)")
var ErrPublicNetwork = errors.New("public network infos are missing")

type Cluster struct {
	pulumi.ResourceState
	ClusterArgs ClusterArgs         `pulumi:"ClusterArgs"`
	Kubeconfig  pulumi.StringOutput `pulumi:"Kubeconfig"`
	SSHKey      pulumi.StringOutput `pulumi:"SSHKey"`
}

type ClusterArgs struct {
	AdditionalPorts    []int
	MachineFlavor      string
	NodeCount          int
	VolumeSize         int
	MachineImage       string
	MachineUser        string
	PrivateNetworkName string
	Public             bool
	PublicIPPool       string
	PublicNetworkName  string
	PublicNetworkID    string
}

// NewCluster is the pulumi program to create a new k3s cluster on top of OpenStack.
func NewCluster(ctx *pulumi.Context, name string, args *ClusterArgs, opts ...pulumi.ResourceOption) (*Cluster, error) {
	cluster := &Cluster{}
	err := ctx.RegisterComponentResource("pkg:k3s:Cluster", name, cluster, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(cluster))

	keyPair, err := compute.NewKeypair(ctx, name, &compute.KeypairArgs{}, opts...)
	if err != nil {
		return nil, err
	}

	secGroupID, err := setupSecurityGroup(ctx, name, args.AdditionalPorts, opts...)
	if err != nil {
		return nil, err
	}

	if !args.Public && args.PrivateNetworkName == "" {
		return nil, ErrPrivateNetwork
	}

	if args.Public &&
		(args.PublicIPPool == "" ||
			args.PublicNetworkName == "" ||
			args.PublicNetworkID == "") {
		return nil, ErrPublicNetwork
	}

	networkName := pulumi.String(args.PrivateNetworkName).ToStringOutput()
	if args.Public {
		networkName, err = setupPublicNetworking(ctx, name, args, opts...)
		if err != nil {
			return nil, err
		}
	}

	if args.NodeCount <= 0 {
		return nil, ErrInvalidNodeCount
	}

	image, err := images.LookupImage(ctx, &images.LookupImageArgs{
		Name: pulumi.StringRef(args.MachineImage),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImageNotFound, err)
	}
	if image == nil {
		return nil, ErrImageNotFound
	}

	var (
		blockDevices compute.InstanceBlockDeviceArray
		imageName    = pulumi.StringPtr(args.MachineImage)
	)
	if args.VolumeSize > 0 {
		blockDevices = compute.InstanceBlockDeviceArray{compute.InstanceBlockDeviceArgs{
			Uuid:                pulumi.String(image.Id), // id from image
			SourceType:          pulumi.String("image"),
			DestinationType:     pulumi.String("volume"),
			DeleteOnTermination: pulumi.BoolPtr(true),
			VolumeSize:          pulumi.IntPtr(args.VolumeSize),
		}}
		imageName = nil
	}

	instanceAddresses := make([]pulumi.StringOutput, 0, args.NodeCount)
	for i := 0; i < args.NodeCount; i++ {
		resourceName := fmt.Sprintf("%s-node-%d", name, i)
		instance, err := compute.NewInstance(ctx, resourceName, &compute.InstanceArgs{
			FlavorName:     pulumi.String(args.MachineFlavor),
			KeyPair:        keyPair.Name,
			ImageName:      imageName,
			SecurityGroups: pulumi.ToStringArrayOutput([]pulumi.StringOutput{secGroupID}),
			Networks: compute.InstanceNetworkArray{
				compute.InstanceNetworkArgs{Name: networkName},
			},
			BlockDevices: blockDevices,
		}, opts...)
		if err != nil {
			return nil, err
		}

		address := instance.AccessIpV4
		if args.Public {
			address, err = setupFIP(ctx, resourceName, args.PublicIPPool, instance.ID(), opts...)
			if err != nil {
				return nil, err
			}
		}

		instanceAddresses = append(instanceAddresses, address)
	}

	masterNodeAddress := instanceAddresses[0]

	var workerNodeAddresses []pulumi.StringOutput
	if len(instanceAddresses) > 1 {
		workerNodeAddresses = instanceAddresses[1:]
	}

	connectionArgsFor := func(host pulumi.StringInput) *remote.ConnectionArgs {
		return &remote.ConnectionArgs{
			Host:       host,
			User:       pulumi.String(args.MachineUser),
			PrivateKey: keyPair.PrivateKey,
			Port:       pulumi.Float64(sshPort),
		}
	}

	kubeconfig, token, err := installK3sMaster(ctx, name, connectionArgsFor(masterNodeAddress), opts...)
	if err != nil {
		return nil, err
	}

	for i, nodeAddress := range workerNodeAddresses {
		if err := installK3sWorker(
			ctx, strconv.Itoa(i), masterNodeAddress, token, connectionArgsFor(nodeAddress), opts...,
		); err != nil {
			return nil, err
		}
	}

	cluster.Kubeconfig = kubeconfig
	cluster.SSHKey = keyPair.PrivateKey

	ctx.RegisterResourceOutputs(cluster, pulumi.Map{
		"Kubeconfig": cluster.Kubeconfig,
		"SSHKey":     cluster.SSHKey,
	})

	return cluster, nil
}

func setupSecurityGroup(
	ctx *pulumi.Context,
	name string,
	additionalPorts []int,
	opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	secGroup, err := networking.NewSecGroup(ctx, name, &networking.SecGroupArgs{
		Description: pulumi.Sprintf("sec group for kindacool cluster %s", name),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	secGroupRules := map[int]string{
		sshPort: "ssh",
		6443:    "kube-apiserver",
		443:     "https",
		80:      "http",
	}

	for _, port := range additionalPorts {
		secGroupRules[port] = "additional-port"
	}

	for port, desc := range secGroupRules {
		_, err = networking.NewSecGroupRule(ctx, fmt.Sprintf("%s-%s-%d", name, desc, port), &networking.SecGroupRuleArgs{
			Direction:       pulumi.String("ingress"),
			SecurityGroupId: secGroup.ID().ToStringOutput(),
			PortRangeMin:    pulumi.IntPtr(port),
			PortRangeMax:    pulumi.IntPtr(port),
			Description:     pulumi.StringPtr(desc),
			Ethertype:       pulumi.String("IPv4"),
			Protocol:        pulumi.StringPtr("tcp"),
		}, opts...)
		if err != nil {
			return pulumi.StringOutput{}, err
		}
	}

	return secGroup.ID().ToStringOutput(), nil
}

func setupPublicNetworking(ctx *pulumi.Context, name string, args *ClusterArgs, opts ...pulumi.ResourceOption) (pulumi.StringOutput, error) {
	network, err := networking.NewNetwork(ctx, name, &networking.NetworkArgs{
		AdminStateUp: pulumi.Bool(true),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	subnet, err := networking.NewSubnet(ctx, name, &networking.SubnetArgs{
		Description: pulumi.String("subnet for all k3s cluster nodes"),
		NetworkId:   network.ID(),
		Cidr:        pulumi.String("10.0.0.0/16"),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	// TODO(brumhard): replace this with actual openstack client since that only requires the network name and not the id as well
	// (at least it should since `openstack network show` does)
	externalNet, err := networking.GetNetwork(ctx, args.PublicNetworkName, pulumi.ID(args.PublicNetworkID), nil, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	router, err := networking.NewRouter(ctx, name, &networking.RouterArgs{
		AdminStateUp:      pulumi.Bool(true),
		ExternalNetworkId: externalNet.ID(),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	_, err = networking.NewRouterInterface(ctx, name, &networking.RouterInterfaceArgs{
		RouterId: router.ID(),
		SubnetId: subnet.ID(),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	return network.Name, nil
}

func setupFIP(
	ctx *pulumi.Context, name string, ipPool string, instanceID pulumi.IDOutput, opts ...pulumi.ResourceOption,
) (pulumi.StringOutput, error) {
	fip, err := networking.NewFloatingIp(ctx, name, &networking.FloatingIpArgs{
		Description: pulumi.String("floating ip for the k3s cluster master node"),
		Pool:        pulumi.String(ipPool),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	_, err = compute.NewFloatingIpAssociate(ctx, name, &compute.FloatingIpAssociateArgs{
		FloatingIp: fip.Address,
		InstanceId: instanceID,
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	return fip.Address, nil
}

func installK3sMaster(
	ctx *pulumi.Context,
	name string,
	connectionArgs *remote.ConnectionArgs,
	opts ...pulumi.ResourceOption,
) (kubeconfig pulumi.StringOutput, token pulumi.StringOutput, err error) {
	installer, err := remote.NewCommand(ctx, "k3s", &remote.CommandArgs{
		// TODO: make channel/version configurable
		Create: pulumi.Sprintf(
			`curl -sfL https://get.k3s.io | \
				INSTALL_K3S_EXEC='server --tls-san="%s"' INSTALL_K3S_CHANNEL="stable" sh -`,
			connectionArgs.Host,
		),
		// TODO check if k3s installed?
		Update:     pulumi.String("echo 'just chilling'"),
		Connection: connectionArgs,
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, pulumi.StringOutput{}, err
	}

	tokenRetriever, err := remote.NewCommand(ctx, "extract-token", &remote.CommandArgs{
		Create:     pulumi.String("sudo cat /var/lib/rancher/k3s/server/node-token"),
		Connection: connectionArgs,
	}, append(opts, pulumi.DependsOn([]pulumi.Resource{installer}))...)
	if err != nil {
		return pulumi.StringOutput{}, pulumi.StringOutput{}, err
	}

	token = tokenRetriever.Stdout.ApplyT(func(tokenWithWhitespace string) string {
		return strings.TrimSpace(tokenWithWhitespace)
	}).(pulumi.StringOutput)

	kubeconfigRetriever, err := remote.NewCommand(ctx, "extract-kubeconfig", &remote.CommandArgs{
		Create:     pulumi.String("sudo cat /etc/rancher/k3s/k3s.yaml"),
		Connection: connectionArgs,
	}, append(opts, pulumi.DependsOn([]pulumi.Resource{installer}))...)
	if err != nil {
		return pulumi.StringOutput{}, pulumi.StringOutput{}, err
	}

	kubeconfig = pulumi.All(kubeconfigRetriever.Stdout, connectionArgs.Host).ApplyT(func(args []interface{}) string {
		kubeconfigReplacer := strings.NewReplacer(
			"127.0.0.1", args[1].(string),
			"localhost", args[1].(string),
			"default", fmt.Sprintf("kindacool-%s", name),
		)

		return kubeconfigReplacer.Replace(args[0].(string))
	}).(pulumi.StringOutput)

	return kubeconfig, token, nil
}

func installK3sWorker(
	ctx *pulumi.Context,
	name string,
	masterAddress pulumi.StringOutput,
	masterToken pulumi.StringOutput,
	connectionArgs *remote.ConnectionArgs,
	opts ...pulumi.ResourceOption,
) error {
	_, err := remote.NewCommand(ctx, fmt.Sprintf("k3s-worker-%s", name), &remote.CommandArgs{
		Create: pulumi.Sprintf(
			`curl -sfL https://get.k3s.io | \
				K3S_URL=https://%s:6443 K3S_TOKEN=%s sh -`,
			masterAddress,
			masterToken,
		),
		Update:     pulumi.String("echo 'just chilling'"),
		Connection: connectionArgs,
	}, opts...)
	if err != nil {
		return err
	}

	return nil
}
