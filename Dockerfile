FROM golang:1.18-alpine3.15 as builder

ENV CGO_ENABLED 0
ENV GOPRIVATE=github.com/dedovvlad/

ARG SWAGGER_PLATFORM
ARG VERSION
ARG BUILD_TIME
ARG APP_NAME
ARG SERVICE_NAME

WORKDIR ${GOPATH}/src/${GOPRIVATE}${VERSION}
RUN PATH=${PATH}:${GOPATH}

COPY . .

RUN apk add git make curl tzdata postgresql && make install-tools &&\
    go install github.com/pressly/goose/v3/cmd/goose@v3.6.1

RUN go build -ldflags "-s -w -X ${GOPRIVATE}${VERSION}/internal/version.VERSION=${VERSION} -X ${GOPRIVATE}${VERSION}/internal/version.BUILD_TIME=${BUILD_TIME} -X gitlab.com/b978/orders/internal/version.SERVICE_NAME=${APP_NAME}" \
    -installsuffix 'static' -o /bin/${VERSION} ./cmd

ENTRYPOINT []
