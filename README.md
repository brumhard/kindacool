# [kinda](https://www.urbandictionary.com/define.php?term=kinda)[cool](https://www.urbandictionary.com/define.php?term=Cool) - `KinD` `a`lternative that is - well... - kinda `cool`

<div align="center">
  <img src="docs/kindacool.png" width="70%">
</div>

`kindacool` is a CLI to quickly setup new Kubernetes (k3s) clusters on OpenStack. It uses [Pulumi Automation API](https://www.pulumi.com/docs/guides/automation-api/) in the background.

## Usage

<details>
  <summary><h3>Installation</h3></summary>

#### From source

If you have Go 1.16+, you can directly install as following:

```shell
go install github.com/brumhard/kindacool/cmd/kindacool@latest
```

> Based on your go configuration the `kindacool` binary can be found in `$GOPATH/bin` or `$HOME/go/bin` in case `$GOPATH` is not set.
> Make sure to add the respective directory to your `$PATH`.
> [For more information see go docs for further information](https://golang.org/ref/mod#go-install). Run `go env` to view your current configuration.

#### From the released binaries

Download the desired version for your operating system and processor architecture from the [releases](https://github.com/brumhard/kindacool/-/releases).
Make the file executable and place it in a directory available in your `$PATH`.

#### OCI Images

OCI images are also available. They contain only the binary `kindacool`, so there's no shell included.

```shell
docker run -it --rm ghcr.io/brumhard/kindacool --help
```

</details>

### Preconditions

#### Pulumi

Pulumi's automation API is used to create all the needed resources in OpenStack and also install k3s. It manages all the state necessary for these operations.

The automation API requires the pulumi CLI.
Install it from any source as described [here](https://www.pulumi.com/docs/get-started/install/).

Afterwards do `pulumi login` (have a look at the [docs](https://www.pulumi.com/docs/reference/cli/pulumi_login/) for all the login options) to connect to the state backend.

#### OpenStack credentials

To connect to OpenStack the common OpenStack environment variables are used. These are usually stored in a `.openrc` file.

Visit the [OpenStack docs](https://docs.openstack.org/newton/user-guide/common/cli-set-environment-variables-using-openstack-rc.html) for more information.

Now do `source /path/to/.openrc` to set all the environment variables.

### Setup your first cluster

> Be sure to first follow the steps described in [preconditions](#preconditions).

To create a new cluster do

```shell
kindacool cluster create
```

This will take around a minute depending on how long it takes to create everything in OpenStack.

After the cluster is created successfully the kubeconfig is written to `~/.kube/kindacool-kindacool.yaml` by default.

You can use the following command to set this path in `$KUBECONFIG`.

```shell
eval `kindacool cluster kubeconfig --export`
```

Now go ahead and test the connection with

```shell
kubectl get pods -A
```

To find docs on all available commands either run `kindacool --help` or visit the [docs](docs/cmd/kindacool.md).

### Misc

```shell
# cancel current action
pulumi cancel <user>/kindacool/<cluster>

# destroy all clusters
kindacool cluster ls |
 while read cluster; do
  if [ "$cluster" = "" ]; then continue; fi
  kindacool cluster destroy --name "$cluster";
 done
```

## Development & Release

> If you have nix installed you can use the flake.nix file to get everything you need.

This project uses [earthly](https://earthly.dev/) for all development and also for CI.

Since it uses containers behind the scenes you need some container runtime like docker or podman installed.
Run `earthly bootstrap` to make sure everything is ready.

The most important earthly targets are:

* Run all linters and tests: `earthly +lint`
* Generate docs: `earthly +generate`
* Trigger a new release: `earthly +tag-release`

If any of the targets do not work as expected add the `-i` flag to run in interactive mode and spawn a shell into the container that is running the step.

### Commit Messages

The commit messages used in this repo should follow [conventional commits style](https://www.conventionalcommits.org/en/v1.0.0/).
This is used by the release tooling to automatically calculate new version tags.

### Releases

To create a new release do:

```shell
earthly +tag-release
```

This will calculate a new semantic version based on the commit messages since the previous version tag and push the newly created tag if necessary.
This should then trigger a pipeline.
If that is not the case you can also run the release locally.

```shell
export GITHUB_TOKEN="xxxxxx"
earthly --push --secret GITHUB_TOKEN +release
```
