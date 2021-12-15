FROM alpine
RUN apk add --update --no-cache ca-certificates
ADD cluster-registry-client /bin/client
USER nobody
ENTRYPOINT ["/bin/client"]
