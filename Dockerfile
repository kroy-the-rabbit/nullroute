FROM golang:1.24 AS builder
WORKDIR /src
ARG TARGETOS
ARG TARGETARCH
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/nullroute ./cmd/nullroute
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/gobgp github.com/osrg/gobgp/v3/cmd/gobgp

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/nullroute /usr/local/bin/nullroute
COPY --from=builder /out/gobgp /usr/local/bin/gobgp
ENTRYPOINT ["/usr/local/bin/nullroute"]
