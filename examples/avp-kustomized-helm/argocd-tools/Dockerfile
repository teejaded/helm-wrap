FROM golang:1.16 as builder

WORKDIR /src/extract-kustomize
COPY go.* .
COPY extract-kustomize.go .
RUN go build -o /extract-kustomize .

FROM alpine:3.14.3

ENV HELM_WRAP_URL=https://github.com/teejaded/helm-wrap/releases/download/20211029-2/helm-wrap_20211029-2_linux_amd64.tar.gz
ENV HELM_3_URL=https://get.helm.sh/helm-v3.6.0-linux-amd64.tar.gz
ENV HELM_2_URL=https://get.helm.sh/helm-v2.17.0-linux-amd64.tar.gz
ENV AVP_URL=https://github.com/IBM/argocd-vault-plugin/releases/download/v1.5.0/argocd-vault-plugin_1.5.0_linux_amd64

WORKDIR /tools

COPY --from=builder extract-kustomize ./
COPY entrypoint.sh /entrypoint.sh
RUN set -x; \
	wget -qO argocd-vault-plugin $AVP_URL && \
	chmod +x argocd-vault-plugin && \
	wget -qO- $HELM_WRAP_URL | tar -xvzf - && \
	wget -qO- $HELM_3_URL | tar -xvzf - && \
	mv linux-amd64/helm ./helm3 && \
	wget -qO- $HELM_2_URL | tar -xvzf - && \
	mv linux-amd64/helm ./helm2 && \
	rm -rf linux-amd64

ENTRYPOINT ["/entrypoint.sh"]
