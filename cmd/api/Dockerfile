FROM alpine
RUN apk add --update --no-cache ca-certificates
ADD cluster-registry-api /bin/api
USER nobody
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
