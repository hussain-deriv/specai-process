FROM --platform=$BUILDPLATFORM golang:1-alpine AS build
ARG TARGETARCH
ARG BUILDPLATFORM
ARG APP_VERSION
WORKDIR /build
COPY . /build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags "-X main.version=$APP_VERSION" -o /digitalcallput cmd/digitalcallput/main.go

FROM alpine
RUN adduser -D -H nonroot
USER nonroot:nonroot
COPY --from=build /digitalcallput /digitalcallput
ENTRYPOINT ["/digitalcallput"]
