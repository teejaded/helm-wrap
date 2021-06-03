# Helm Wrap

Helm Wrap is a Helm wrapper which processes yaml values files and helm output. It passes values files through named pipes incase you are decrypting them.

This tool is intended to be used with [ArgoCD's Helm feature](https://argoproj.github.io/argo-cd/user-guide/helm/).  It enables you to pre-process values or post-process helm output without using a custom plugin.

## Installation

### Prerequisites

Helm is needed for Helm Wrap to work. Follow the instructions [here](https://helm.sh/docs/intro/install/) to install it.

### Getting Helm Wrap binary

#### Helm Wrap releases

Helm Wrap released binaries can be downloaded from [GitHub](https://github.com/teejaded/helm-wrap/releases).

#### Building from sources

Helm Wrap can be built using the `go build` command.

### Deploying Helm Wrap

* Rename helm to _helm.
* Rename helm2 to _helm2 
* Add the `helm-wrap` binary with the names `helm` and `helm2`. 

You can do this using an init container or by building custom images.  Here is an example using the [argo-cd helm chart](https://github.com/argoproj/argo-helm/tree/master/charts/argo-cd).

```
repoServer:
  volumes:
  - name: custom-tools
    emptyDir: {}

  volumeMounts:
    - mountPath: /usr/local/bin/_helm2
      name: custom-tools
      subPath: helm-v2
    - mountPath: /usr/local/bin/_helm
      name: custom-tools
      subPath: helm-v3

    # mount helm-wrap as helm and helm2
    - mountPath: /usr/local/bin/helm
      name: custom-tools
      subPath: helm-wrap
    - mountPath: /usr/local/bin/helm2
      name: custom-tools
      subPath: helm-wrap

  initContainers:
    - name: download-tools
      image: alpine:latest
      imagePullPolicy: Always
      env:
        - name: HELM_SOPS_URL
          value: "https://github.com/teejaded/helm-sops/releases/download/20201103-2/helm-sops_20201103-2_linux_amd64.tar.gz"
        - name: HELM_3_URL
          value: "https://get.helm.sh/helm-v3.4.2-linux-amd64.tar.gz"
        - name: HELM_2_URL
          value: "https://storage.googleapis.com/kubernetes-helm/helm-v2.17.0-linux-amd64.tar.gz"
      command: [sh, -c]
      args:
        - >-
          set -x;
          cd /custom-tools &&
          wget -qO- $HELM_SOPS_URL | tar -xvzf - &&
          wget -qO- $HELM_3_URL | tar -xvzf - &&
          mv linux-amd64/helm /custom-tools/helm-v3 &&
          wget -qO- $HELM_2_URL | tar -xvzf - &&
          mv linux-amd64/helm /custom-tools/helm-v2
      volumeMounts:
        - mountPath: /custom-tools
          name: custom-tools
```

## Usage

Create a config json that processes your yaml.  The config consists of an array of actions that are executed in order.

### transform-values action

This action calls your command for each helm values file found in the arguments. Stdout is captured and written to a named pipe.

The values file path is subsituted for `{}`.

There is an optional "filter" parameter which will check if a json path exists before running your command.


### shell-exec

This action runs the command using `/bin/sh -c`.  It adds an environment variable `HELM` that contains the correct binary and arguments.

## Example Configs

This configuration replicates the functionality of Camptocamp's helm-sops

```json
[
	{
		"action": "transform-values",
		"filter": "$.sops.lastmodified",
		"command": "sops -d {}"
	},
	{
		"action": "shell-exec",
		"command": "$HELM"
	}
]
```

kustomized-helm without a plugin

```json
[
	{
		"action": "shell-exec",
		"command": "$HELM > all.yaml; kustomize build $ARGOCD_APP_SOURCE_TARGET_REVISION"
	}
]
```

argocd-vault-plugin

```json
[
	{
		"action": "shell-exec",
		"command": "$HELM > all.yaml; argocd-vault-plugin generate all.yaml"
	}
]
```