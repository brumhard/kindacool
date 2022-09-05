package app

import (
	"fmt"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
)

// KubeDir returns the path to the directory to safe kubeconfigs in.
// It also ensures that the directory exists.
func KubeDir() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	kubedir := path.Join(home, ".kube")
	if err := os.MkdirAll(kubedir, os.ModeDir); err != nil {
		return "", err
	}

	return kubedir, nil
}

func KubeconfigFile(clusterName string) (string, error) {
	kubeDir, err := KubeDir()
	if err != nil {
		return "", err
	}

	return path.Join(kubeDir, fmt.Sprintf("kindacool-%s.yaml", clusterName)), nil
}
