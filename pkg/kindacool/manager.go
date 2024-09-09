package kindacool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/brumhard/kindacool/pkg/k3s"
	"github.com/gophercloud/gophercloud/openstack"

	"github.com/pulumi/pulumi/pkg/v3/backend/httpstate"
	pkgWorkspace "github.com/pulumi/pulumi/pkg/v3/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// TODO: make configurable
const (
	defaultProjectName = "kindacool"
	OutputKubeconfig   = "kubeconfig"
	OutputSSHKey       = "sshKey"
)

var (
	ErrPulumiNotInPath   = errors.New("pulumi executable not found in $PATH")
	ErrUnauthorized      = errors.New(".openrc env vars for openstack could not be found")
	ErrOutputUnavailable = errors.New("output could not be found")
)

type Manager struct {
	Options GlobalOptions
	Logger  *log.Logger
}

func EnsureEnvironment(ctx context.Context) error {
	pul := exec.CommandContext(ctx, "pulumi", "--help")
	if err := pul.Run(); err != nil {
		return ErrPulumiNotInPath
	}

	_, err := httpstate.NewLoginManager().Current(
		ctx,
		httpstate.DefaultURL(pkgWorkspace.Instance),
		false, false,
	)
	if err != nil {
		return err
	}

	_, err = openstack.AuthOptionsFromEnv()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	return nil
}

func (m *Manager) Run(ctx context.Context, args *k3s.ClusterArgs) error {
	// inline pulumi program
	deployFunc := func(ctx *pulumi.Context) error {
		cluster, err := k3s.NewCluster(ctx, m.Options.Name, args)
		if err != nil {
			return err
		}

		ctx.Export(OutputKubeconfig, pulumi.ToSecret(cluster.Kubeconfig))
		ctx.Export(OutputSSHKey, pulumi.ToSecret(cluster.SSHKey))

		return nil
	}

	stackName := m.Options.Name

	m.Logger.Printf("Creating/using stack %q\n", stackName)
	// TODO(brumhard): probably can add auto.Project() and then configure a custom backend there
	// to not require the user to login and just use file backend in some ~/.kindacool dir maybe.
	s, err := auto.UpsertStackInlineSource(ctx, stackName, defaultProjectName, deployFunc)
	if err != nil {
		return fmt.Errorf("failed to get/create stack: %w", err)
	}

	m.Logger.Println("Installing required pulumi plugins")
	if err := EnsurePlugins(ctx, s.Workspace()); err != nil {
		return err
	}

	m.Logger.Println("Checking for existing resources")
	_, err = s.Refresh(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh stack: %w", err)
	}

	m.Logger.Println("Creating/updating required resources")
	stdoutStreamer := optup.ProgressStreams(io.Discard)
	if m.Options.Verbose {
		stdoutStreamer = optup.ProgressStreams(m.Logger.Writer())
	}

	_, err = s.Up(ctx, stdoutStreamer)
	if err != nil {
		return fmt.Errorf("failed to update stack: %w", err)
	}

	m.Logger.Println("Successfully created your fresh k3s cluster!")

	return nil
}

func (m *Manager) Destroy(ctx context.Context) error {
	project := workspace.Project{
		Name:    tokens.PackageName(defaultProjectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	}

	w, err := auto.NewLocalWorkspace(ctx, auto.Project(project))
	if err != nil {
		return err
	}

	m.Logger.Println("Looking for cluster")
	stack, err := auto.SelectStack(ctx, m.Options.Name, w)
	if err != nil {
		if auto.IsSelectStack404Error(err) {
			m.Logger.Printf("No cluster with name %q could be found\n", m.Options.Name)
			return nil
		}

		return err
	}

	m.Logger.Println("Installing required pulumi plugins")
	if err := EnsurePlugins(ctx, w); err != nil {
		return err
	}

	m.Logger.Println("Destroying resources")
	stdoutStreamer := optdestroy.ProgressStreams(io.Discard)
	if m.Options.Verbose {
		stdoutStreamer = optdestroy.ProgressStreams(m.Logger.Writer())
	}

	_, err = stack.Destroy(ctx, stdoutStreamer)
	if err != nil {
		return err
	}

	m.Logger.Println("Removing stack")
	w.RemoveStack(ctx, m.Options.Name)

	m.Logger.Println("Your cluster was destroyed successfully!")

	return nil
}

func (m *Manager) List(ctx context.Context) ([]string, error) {
	project := workspace.Project{
		Name:    tokens.PackageName(defaultProjectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	}

	w, err := auto.NewLocalWorkspace(ctx, auto.Project(project))
	if err != nil {
		return nil, err
	}

	stacks, err := w.ListStacks(ctx)
	if err != nil {
		return nil, err
	}

	clusters := make([]string, 0, len(stacks))
	for _, stack := range stacks {
		clusters = append(clusters, stack.Name)
	}

	return clusters, nil
}

// FetchOutput gets an output from the current stack.
// The available outputs are defined as consts.
func (m *Manager) FetchOutput(ctx context.Context, outputKey string) (string, error) {
	project := workspace.Project{
		Name:    tokens.PackageName(defaultProjectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	}

	w, err := auto.NewLocalWorkspace(ctx, auto.Project(project))
	if err != nil {
		return "", err
	}

	stack, err := auto.SelectStack(ctx, m.Options.Name, w)
	if err != nil {
		return "", err
	}

	outputs, err := stack.Outputs(ctx)
	if err != nil {
		return "", err
	}

	kubeconfig, ok := outputs[outputKey].Value.(string)
	if !ok {
		return "", fmt.Errorf("%s: %w", outputKey, ErrOutputUnavailable)
	}

	return kubeconfig, nil
}
