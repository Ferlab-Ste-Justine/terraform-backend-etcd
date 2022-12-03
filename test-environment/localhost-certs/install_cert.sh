#!/usr/bin/env sh

cp certs/local_ca.crt /usr/local/share/ca-certificates
update-ca-certificates