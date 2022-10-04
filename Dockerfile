FROM golang:1.18-alpine3.15 as builder

ENV CGO_ENABLED 0
ENV GOPRIVATE=github.com/dedovvlad/*

ARG SWAGGER_PLATFORM
ARG VERSION
ARG BUILD_TIME
ARG APP_NAME


RUN PATH=${PATH}:${GOPATH}

COPY . .

RUN apk add git make curl tzdata postgresql && make install-tools &&\
    go install github.com/pressly/goose/v3/cmd/goose@v3.6.1 &&\
    go install github.com/go-testfixtures/testfixtures/v3/cmd/testfixtures@v3.6.1

RUN go build -ldflags "-s -w -X github.com/dedovvlad/${PROJECT}/internal/version.VERSION=${VERSION} -X github.com/dedovvlad/${PROJECT}/internal/version.BUILD_TIME=${BUILD_TIME} -X gitlab.com/b978/orders/internal/version.SERVICE_NAME=${APP_NAME}" \
    -installsuffix 'static' -o /bin/orders_api ./cmd/api
RUN go build -ldflags "-s -w -X github.com/dedovvlad/${PROJECT}/internal/version.VERSION=${VERSION} -X github.com/dedovvlad/${PROJECT}/version.BUILD_TIME=${BUILD_TIME} -X gitlab.com/b978/orders/internal/version.SERVICE_NAME=${APP_NAME}" \
    -installsuffix 'static' -o /bin/orders_async ./cmd/async

ENTRYPOINT []

FROM alpine:3.15 as production

RUN apk add tzdata
COPY --from=builder /bin/* /bin/
COPY --from=builder /go/bin/* /bin/
COPY database /app/database

USER nobody
ENTRYPOINT [ "sh", "-c", "/bin/orders_api" ]
