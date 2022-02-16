FROM alpine
RUN apk add --update --no-cache ca-certificates
ADD cluster-registry-apiserver /bin/apiserver
USER nobody
EXPOSE 8080
ENTRYPOINT ["/bin/apiserver"]
