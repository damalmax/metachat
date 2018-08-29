FROM golang:alpine as builder
RUN apk add git && apk add ca-certificates
RUN adduser -D -g '' appuser
RUN mkdir /build
COPY . /build/
WORKDIR /build
RUN CGO_ENABLED=0 go build -o main .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /build/main /app/
USER appuser
WORKDIR /app
ENTRYPOINT ["./main"]