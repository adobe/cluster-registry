FROM alpine
RUN apk add --update --no-cache ca-certificates
ADD cluster-registry-sync-producer /bin/sync-producer
USER nobody
ENTRYPOINT ["/bin/sync-producer"]
