# About

Hashicorp has deprecated the etcd backend in Terraform.

Even prior to the deprecation, their support for etcd as a backend was not very good as it bundled the entire terraform state in a single etcd key (making it more likely that etcd's 1.5MB key size limit would impact the maximum possible scope of your terraform orchestrations) instead of distributing the state across many keys.

This repo means to remedy both those thigs and provide you with the etcd terraform backend that you deserve by abstracting etcd behind an http server (leveraging terraform's http backend). For why we are doing this, go to the end of the documentation.

# Why Etcd as a Terraform Backend

I believe the appeal is mostly for those wanting to take advantage of some of the advances in the cloud, but with an on-premise setup.

If you are on the cloud, the answer is that you should probably go with an existing backend for your cloud provider if there is one.

If you are on-prem and are regulated, you are most likely limited to the following backends out of the box:
- **Kubernetes**: Heavy to setup and operate, you might not wish to run persistent states on kubernetes if you are 1-2 people maintaining it in addition to everything else. Treating it as "throwaway immutable infra" make operating it simpler. Also, the backend would be limited to 1mb of size anyways
- **Postgres**: If you are running an ha setup with Patroni, it is not exactly feather weight... a trio of postgres servers, a trio of etcd servers, bunch of load balancers... as far as our near-term postgres expertize goes, there will also be some downtime during version upgrades...
- **Consul**: Great if you are already using it. If you aren't and have no other need for it, do you want the cognitive overhead of having to manage yet another stateful component in your stack just for your terraform states?

Etcd on the other end is only 3 nodes for a minimal robust ha cluster and pretty straightfoward to operate as far as stateful components go (assuming you read the documentation of course). If you aren't running at Google scale (which you won't as a small team), it will be more than performant enough for your needs (already very optimized for kubernetes). It is also dependable and battle-tested to hell and back. Plus odds are you're gonna need it for a bunch of stuff anyways (kubernetes, minio, possibly patroni and vault, maybe even as a store for your dns...).

All that's missing is a decent backend integration for terraform and this project aims to fill that void for us, leveraging the http backend integration: https://www.terraform.io/language/settings/backends/http

# Usage

Your backend declaration will look like this (yes, encoding part of the resource url in the query string for non-GET is a little weird, but it allows you to easily url-encode path values and both terraform and the Gin framework roll with it):

```
terraform {
  backend "http" {
    update_method = "PUT"
    address = "<http|https>://<url>:<port>/state?state=${urlencode("<state etcd prefix>")}"
    lock_method = "PUT"
    lock_address = "<http|https>://<url>:<port>/lock?state=${urlencode("<state etcd prefix>")}&lease_ttl=<deadline to release lock>"
    unlock_method = "DELETE"
    unlock_address = "<http|https>://<url>:<port>/lock?state=${urlencode("<state etcd prefix>")}"
  }
}
```

Then, you will have a configuration file for the backend that looks like this:

```
server:
  port: <port to bind the http server on>
  address: "<address to bind the http server on>"
etcd_client:
  endpoints: "<etcd1 url>:<etcd1 port>,<etcd2 url>:<etcd2 port>,..."
  connection_timeout: "<connection timeout on etcd as golang duration string. Put at least a minute>"
  request_timeout: "<request timeout on etcd as golang duration string. Put at least a minute>"
  retries: <number of times to retry>
  auth:
    ca_cert: "<path to etcd ca cert>"
    client_cert: "<path to the client cert to authentify with etcd>"
    client_key: "<path to the client private key to authentify with etcd>"
```