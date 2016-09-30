FROM alpine:latest

RUN apk --update upgrade && \
    apk add curl ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

COPY dockervault ./app/dockervault

ENV VAULT_TOKEN no-token
ENV VAULT_ADDR http://active.vault.service.consul:8200
ENV DV_FILE_PATH /tmp/init-token
ENV DV_DOCKER_HOST /var/run/docker.sock
ENV DV_MAPPING_KEY secret/dockervault

ENTRYPOINT ["/app/dockervault"]