#FROM golang:1.9-alpine as build-deps
#
#ARG PROM_VERSION
#ARG PROM_RULES_VERSION
#ARG BOMB_SQUAD_VERSION
#
#ENV CGO_ENABLED=0
#
#WORKDIR /go/src/github.com/Fresh-Tracks/bomb-squad
#COPY . .
#
#RUN go build -a \
#      -ldflags="-X main.promVersion=$PROM_VERSION -X main.promRulesVersion=$PROM_RULES_VERSION -X main.version=$BOMB_SQUAD_VERSION" \
#      -o /bin/bs

## Final artifact
FROM alpine:3.7

COPY ./bin/bs /bin/bs
COPY ./prom_rules.yaml /etc/bomb-squad/rules.yaml

ENTRYPOINT ["/bin/bs"]
