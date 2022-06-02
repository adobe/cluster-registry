FROM golang AS build

ADD . /cluster-registry
WORKDIR /cluster-registry

RUN go install test/slt/update.go


FROM alpine

RUN apk update && apk add libc6-compat \
               && apk add bash \
               && apk add curl \
               && apk add jq \
               && rm -rf /var/cache/apk/*

COPY --from=build /go/bin/update /bin/update

RUN chmod +x /bin/update

ENTRYPOINT ["/bin/update"]
