#!/usr/bin/env sh

(cd ../..; go build .; cp terraform-backend-etcd test-environment/backend-server)

ETCD_BACKEND_CONFIG_FILE=config.yml ./terraform-backend-etcd