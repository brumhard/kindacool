## kindacool cluster kubeconfig

Output a cluster's kubeconfig

### Synopsis

The kubeconfig command has to modes of working.

1. By default the command will fetch the cluster's kubeconfig from Pulumi and write it to stdout.
   Generally, the kubeconfig will also be written to ~/.kube/kindacool-<clustername>.yaml after creation.

2. If the --export flag is set it will instead output a shell command to set $KUBECONFIG to the cluster's kubeconfig's default file path.
   This can be used like:
   	$ eval "$(kindacool cluster kubeconfig --export)"


```
kindacool cluster kubeconfig [flags]
```

### Options

```
      --export   Output an export string pointing to the file instead of the kubeconfig itself.
  -h, --help     help for kubeconfig
```

### Options inherited from parent commands

```
  -n, --name string   Name of the cluster to manage. (default "kindacool")
  -v, --verbose       Enable verbose pulumi output.
```

### SEE ALSO

* [kindacool cluster](kindacool_cluster.md)	 - kindacool cluster is the main entrypoint to all cluster management operations

###### Auto generated by spf13/cobra on 3-Feb-2023
