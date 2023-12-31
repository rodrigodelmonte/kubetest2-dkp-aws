/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package deployer implements the kubetest2 kind deployer
package deployer

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/octago/sflags/gen/gpflag"
	"github.com/spf13/pflag"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/types"
)

// Name is the name of the deployer
const Name = "dkp-aws"

var GitTag string

// New implements deployer.New for kind
func New(opts types.Options) (types.Deployer, *pflag.FlagSet) {
	// create a deployer object and set fields that are not flag controlled
	d := &deployer{
		commonOptions: opts,
	}
	// register flags and return
	return d, bindFlags(d)
}

// assert that New implements types.NewDeployer
var _ types.NewDeployer = New

type deployer struct {
	// generic parts
	commonOptions types.Options

	ClusterName      string `flag:"cluster-name" desc:"the cluster name"`
	KuberneteVersion string `flag:"kubernetes-version" desc:"the kubernetes version"`
	Ami              string `flag:"ami" desc:"the Ami ID to use for build and up"`

	KubeconfigPath string `flag:"-"` // populated by Up()
}

func (d *deployer) Up() error {

	args := []string{
		"create",
		"cluster",
		"aws",
		"--cluster-name", d.ClusterName,
		"--ami", d.Ami,
		"--kubernetes-version", d.KuberneteVersion,
		"--with-aws-bootstrap-credentials=true",
		"--self-managed",
	}
	if exitCode := runner("dkp", args, os.Stdout, os.Stderr); exitCode != 0 {
		return errors.New("failed to run dkp create cluster")
	}
	d.KubeconfigPath = d.ClusterName + ".conf"

	return nil
}

func (d *deployer) Down() error {
	command_args := [][]string{
		{
			"create",
			"bootstrap",
			"--with-aws-bootstrap-credentials=true",
		},
		{
			"move",
			"capi-resources",
			"--from-kubeconfig", d.KubeconfigPath,
			"--to-context",
			"kind-konvoy-capi-bootstrapper",
		},
		{
			"delete",
			"cluster",
			"--cluster-name", d.ClusterName,
		},
		{
			"delete",
			"bootstrap",
		},
	}
	for _, args := range command_args {

		if exitCode := runner("dkp", args, os.Stdout, os.Stderr); exitCode != 0 {
			return errors.New("failed to delete dkp cluster")
		}
	}

	return nil
}

func (d *deployer) IsUp() (up bool, err error) {

	config, err := clientcmd.BuildConfigFromFlags("", d.KubeconfigPath)
	if err != nil {
		return false, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return false, err
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status != "True" {
				return false, errors.New("cluster nodes are not ready yet")
			}
		}
	}

	return true, nil
}

func (d *deployer) DumpClusterLogs() error {
	return nil
}

func (d *deployer) Build() error {
	// TODO: build should probably still exist with common options
	return nil
}

func (d *deployer) Kubeconfig() (string, error) {
	if d.KubeconfigPath != "" {
		return d.KubeconfigPath, nil
	}
	if kconfig, ok := os.LookupEnv("KUBECONFIG"); ok {
		return kconfig, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kube", "config"), nil
}

func (d *deployer) Version() string {
	return GitTag
}

// helper used to create & bind a flagset to the deployer
func bindFlags(d *deployer) *pflag.FlagSet {
	flags, err := gpflag.Parse(d)
	if err != nil {
		klog.Fatalf("unable to generate flags from deployer")
		return nil
	}

	flags.AddGoFlagSet(flag.CommandLine)

	return flags
}

// assert that deployer implements types.DeployerWithKubeconfig
var _ types.DeployerWithKubeconfig = &deployer{}

// runner executes commands with the given arguments
func runner(cmd string, args []string, stdout, stderr io.Writer) int {
	cmdArgs := []string{}
	cmdArgs = append(cmdArgs, args...)
	c := exec.Command(cmd, cmdArgs...)

	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
	}
	return 0
}
