FROM alpine
RUN apk add --update --no-cache ca-certificates
ADD cluster-registry-sync-client /bin/sync-client
USER nobody
ENTRYPOINT ["/bin/sync-client"]
