# AVP Kustomized Helm Example

## Introduction

Our team avoids using [ArgoCD custom plugins](https://argo-cd.readthedocs.io/en/stable/user-guide/config-management-plugins/) because we like the feature set [Helm Applications](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/) gives us.  Custom plugins are very basic and lack the features of a Helm Application.

At the same time, there is a missing feature for ArgoCD which should allow for customizable pre/post processing of values and output.  The reason for that is, maintaining chart forks is labor intensive and chart authors don't always want to merge in edge-case pull requests.  Also, for us, manually maintaining per-server encrypted secrets using kubeseal is error prone, labor intensive and does not integrate with the Helm Application feature.

This example shows how we use ArgoCD's Helm Applications with Kustomize and argocd-vault-plugin to resolve all of these issues and reduce toil.

## Installing ArgoCD with helm-wrap, extract-kustomize and argocd-vault-plugin.

We use a prebaked init-container to pass the custom tools to the `argocd-repo-server` container.  This lets us avoid Github rate limiting and the overhead of maintaining a customized `argocd-repo-server` image that we must update for every ArgoCD upgrade.

On startup the init-container copies all of the tools to an emptydir mount. We then use `volumeMounts` to add the binaries to `/usr/local/bin`.

### Customize argocd-vault-plugin authentication

We use Vault's Kubernetes authentication for `argocd-vault-plugin`.  Check the [argocd-vault-plugin documentation](https://ibm.github.io/argocd-vault-plugin/v1.5.0/backends/) to see the available authentication methods and backends.

Change the following section of `argocd-repo-server-values.yaml` to match your setup.

```yaml
    - name: "VAULT_ADDR"
      value: "https://my-vault-server"
    - name: AVP_TYPE
      value: vault
    - name: AVP_AUTH_TYPE
      value: k8s
    - name: AVP_K8S_ROLE
      value: argocd
```

### Install ArgoCD

To install, use your modified values file to install ArgoCD with the custom tools.  Check the [chart documentation](https://github.com/argoproj/argo-helm/tree/master/charts/argo-cd) for more information.

```bash
$ helm repo add argo https://argoproj.github.io/argo-helm
"argo" has been added to your repositories

$ helm install my-release argo/argo-cd -f argocd-repo-server-values.yaml
NAME: my-release
...
```

## helm-wrap config

Included in the values file above we set the `HELMWRAP_CONFIG` environment variable to the following.

```json
[
    {
        "action": "transform-values",
        "filter": "$.Kustomize",
        "command": "extract-kustomize {}"
    },
    {
        "action": "shell-exec",
        "filter": "template",
        "command": "([ ! -f kustomization.yaml ] && $HELM || ($HELM > all.yaml && kustomize build)) | argocd-vault-plugin generate -"
    },
    {
        "action": "shell-exec",
        "command": "$HELM"
    }
]
```

Each of these three actions are explained in detail below.

### Extract Kustomize

The first action pre-processes helm values files.  If the jsonpath `$.Kustomize` exists we will run `extract-kustomize` with the path to the values file.  This binary is a minimal go program, found in the argocd-tools directory, that writes each key from the Kustomize map to a file.

This lets us embed the Kustomize configuration into the Helm values and extract it at runtime.

### Wrapped helm template command

The second action is triggered when ArgoCD runs `helm template`.  This is called when ArgoCD is generating the yaml manifests it will deploy.

If a `kustomization.yaml` file exists we will run the original command `$HELM` and save the output in `all.yaml`.  Then, we call `kustomize build` which will modify the helm output as specified in the `kustomization.yaml` file.

The stdout from kustomize or helm is then piped into `argocd-vault-plugin generate -` which will fill in placeholders using your secrets backend and writes to stdout.

### Fall through helm command

The last action runs the original command from argocd if the helm command did not match any other filter.  Usually this is used when ArgoCD is looking for and downloading helm charts.  You could add additional actions here to support custom helm repositories.

## Use case #1 -- Patching storage classes

The following example shows how we have patched the the storage classes to for the `ibm-block-storage` plugin.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: ibm-block-storage
spec:
  project: ibm-block-storage
  source:
    repoURL: 'https://icr.io/helm/iks-charts'
    targetRevision: 2.1.2
    helm:
      releaseName: ibm-block-storage
      values: |
        Kustomize:
          kustomization.yaml: |
            apiVersion: kustomize.config.k8s.io/v1beta1
            kind: Kustomization
            resources:
            - all.yaml
            patches:
            - patch: |-
                - op: add
                  path: /volumeBindingMode
                  value: WaitForFirstConsumer
              target:
                kind: StorageClass
    chart: ibmcloud-block-storage-plugin
  destination:
    server: 'https://my-k8s-server'
    namespace: kube-system
```

## Use case #2 -- Deploy an out-of-band secret with a chart

It's a fairly common pattern to see a helm chart that expects you to create a secret out-of-band and provide the name in the values.

Here we are using argocd-vault-plugin to provide the `adminPassword` and the values for the secret generated by kustomize.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kube-prometheus-stack
spec:
  destination:
    namespace: monitoring
    server: https://my-k8s-server
  project: kube-prometheus-stack
  source:
    chart: kube-prometheus-stack
    repoURL: https://prometheus-community.github.io/helm-charts
    targetRevision: 19.2.3
    helm:
      releaseName: prometheus-operator
      values: |
        grafana:
          adminPassword: <path:secrets/monitoring/grafana#adminPassword>
          grafana.ini:
            auth.generic_oauth:
              api_url: https://oidc-server/userinfo
              auth_url: https://oidc-server/authorize
              client_id: "$__file{/etc/secrets/auth_generic_oauth/client_id}"
              client_secret: "$__file{/etc/secrets/auth_generic_oauth/client_secret}"
              enabled: "true"
              scopes: openid
              token_url: https://oidc-server/token
          extraSecretMounts:
          - name: auth-generic-oauth-secret-mount
            secretName: auth-generic-oauth-secret
            defaultMode: 0440
            mountPath: /etc/secrets/auth_generic_oauth
            readOnly: true

        Kustomize:
          kustomization.yaml: |
            apiVersion: kustomize.config.k8s.io/v1beta1
            kind: Kustomization
            resources:
            - all.yaml
            secretGenerator:
            - name: auth-generic-oauth-secret
              literals:
              - client_id=<path:secrets/monitoring/grafana#clientID>
              - client_secret=<path:secrets/monitoring/grafana#clientSecret>
```

