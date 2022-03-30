FROM golang AS build

ADD . /cluster-registry
WORKDIR /cluster-registry

RUN go install github.com/rakyll/hey@latest
RUN go install test/performance/heyparser.go


FROM alpine

RUN apk update && apk add libc6-compat \
               && apk add bash \
               && apk add curl \
               && apk add jq \
               && rm -rf /var/cache/apk/*

COPY --from=build /go/bin/hey /bin/hey
COPY --from=build /go/bin/heyparser /bin/heyparser

RUN chmod +x /bin/hey /bin/heyparser

COPY test/performance/scripts/benchmark.sh /script/

ENTRYPOINT ["/script/benchmark.sh"]
