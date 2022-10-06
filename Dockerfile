from golang:1.18-bullseye as builder

ENV CGO_ENABLED=0

WORKDIR /opt

COPY . /opt/

RUN go mod tidy

RUN go build .

FROM scratch

COPY --from=builder /opt/terraform-backend-etcd /terraform-backend-etcd

ENTRYPOINT ["/terraform-backend-etcd"]