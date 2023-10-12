# kubetest2-dkp-aws

> kubetest2-dkp-aws is a proof of concept to execute Kubernetes Conformance tests on AWS clusters deployed using DKP.

## Requirements

* `dkp` CLI
* `kubectl` CLI
* Configured AWS credentials

## Install

```sh
go install sigs.k8s.io/kubetest2/...@latest
go install sigs.k8s.io/kubetest2/kubetest2-tester-ginkgo@latest
go install github.com/rodrigodelmonte/kubetest2-dkp-aws
```

## Running Kubernetes Conformance Tests

```sh
CLUSTER_NAME=kubetest2-rhel-88
AMI_ID=ami-05729346d7b1ec312 # Built with https://github.com/mesosphere/konvoy-image-builder/
KUBERNETES_VERSION=1.27.5
kubetest2 dkp-aws \
    --up \
    --down \
    --cluster-name=${CLUSTER_NAME} \
    --ami=${AMI_ID} \
    --kubernetes-version=${KUBERNETES_VERSION} \
    --test=ginkgo \
    --  \
    --parallel 8 \
    --flake-attempts 2 \
    --test-package-version v${KUBERNETES_VERSION} \
    --focus-regex='\[Conformance\]' | tee e2e.log
```

## References

* <https://github.com/kubernetes-sigs/kubetest2#kubetest2>
