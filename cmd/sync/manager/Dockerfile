FROM alpine
RUN apk add --update --no-cache ca-certificates
ADD cluster-registry-sync-manager /bin/sync-manager
USER nobody
ENTRYPOINT ["/bin/sync-manager"]
