FROM golang AS build

ADD . /cluster-registry
WORKDIR /cluster-registry

RUN go install test/slt/service.go


FROM alpine

RUN apk update && apk add libc6-compat \
               && apk add bash \
               && apk add curl \
               && apk add jq \
               && rm -rf /var/cache/apk/*

COPY --from=build /go/bin/service /bin/sltservice

RUN chmod +x /bin/sltservice

ENTRYPOINT ["/bin/sltservice"]
