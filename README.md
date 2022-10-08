# About

We use etcd v3 quite a bit here for our operations and have wanted better support for etcd as a backend to terraform (and provider, which we also did in another project) for some time.

The deprecation sectre loomed on the official etcd backend for a while and it finally got deprecated in Terraform version 1.3.

Even prior to the deprecation, the support for etcd with the official backend was not very good as it bundled the entire terraform state in a single etcd key (making etcd's 1.5MB key size limit also a limit on the maximum backend size of individual terraform orchestrations) instead of distributing the state across many keys.

This repo means to remedy both those things and provide us with the etcd terraform backend that we feel we deserve by abstracting etcd behind an http server (leveraging terraform's http backend).

# Why Etcd as a Terraform Backend

We believe the appeal is especially strong for small teams like us that have an on-premise setup, that have stringent manpower constraints so we can't comfortably operate every solution under the sun and yet we don't want to operate our still sizeable infra old school, taking advantage of advances like Gitops, infra as code and Kubernetes.

If you are on the cloud, the answer to your terraform backend problem is that you should probably go with an existing backend for your cloud provider if there is one.

If you are on-prem and are regulated, you are most likely limited to the following backends out of the box:
- **Kubernetes**: Heavy to setup and operate, you might not wish to run persistent states on kubernetes if you are 1-2 people maintaining it in addition to everything else. Treating it as "throwaway immutable infra" make operating it simpler. Also, the backend would be limited to 1mb of size anyways
- **Postgres**: If you are running an ha setup with Patroni, it is not exactly feather weight... a trio of postgres servers, a trio of etcd servers, bunch of load balancers... as far as our near-term postgres expertize goes, there will also be some downtime during version upgrades...
- **Consul**: Great if you are already using it. If you aren't and have no other need for it, do you want the cognitive overhead of having to manage yet another stateful component in your stack just for your terraform states?

Etcd on the other hand only requires 3 nodes for a minimal robust ha cluster and is pretty straightfoward to operate as far as stateful components go. If you aren't running at Google scale (which you won't as a small team), it will be more than performant enough for your needs (already very optimized for kubernetes), plus you can defensively segment your worflows in several etdc clusters. Etcd is also well battle-tested, benefitting from the kind of thoroughness that a projet used in almost every kubernetes cluster would have. Odds are strong you will need it for a couple of things anyways (kubernetes, minio, possibly patroni and vault, maybe even as a store for your dns...) so might as well double down on that expertize.

All that is missing for us to leverage etcd for our terraform states is a decent backend integration and this project aims to fill that void, leveraging the http backend integration: https://www.terraform.io/language/settings/backends/http

# Usage

Your backend declaration will look like this (encoding part of the resource url in the query string for non-GET is a little weird, but it allows you to easily url-encode path values and both terraform and the Gin framework roll with it):

```
terraform {
  backend "http" {
    update_method = "PUT"
    address = "<http|https>://<url>:<port>/state?state=<url encoded state etcd prefix>"
    lock_method = "PUT"
    lock_address = "<http|https>://<url>:<port>/lock?state=<url encoded state etcd prefix>&lease_ttl=<deadline to release lock>"
    unlock_method = "DELETE"
    unlock_address = "<http|https>://<url>:<port>/lock?state=<url encoded state etcd prefix>"
    username = "<basic auth username if you use it, else omit>"
    password = "<basic auth password if you use it, else omit>"
  }
}
```

Then, you will have a configuration file for the server that looks like this:

```
server:
  port: <port to bind the http server on>
  address: "<address to bind the http server on>"
  auth:
    ca_cert: "<path to etcd ca cert>"
    client_cert: "<path to the client cert to authentify with etcd>"
    client_key: "<path to the client private key to authentify with etcd>"
    basic_auth: "<path to yaml basic auth file if you want basic auth>"
  tls:
    certificate: <Path to server certificate if you want to use tls>
    key: <Path to server key if you want to use tls>
etcd_client:
  endpoints: 
    - "<etcd1 url>:<etcd1 port>"
    - "<etcd2 url>:<etcd2 port>"
    ...
  connection_timeout: "<connection timeout on etcd as golang duration string. Put at least a minute>"
  request_timeout: "<request timeout on etcd as golang duration string. Put at least a minute>"
  retries: <number of times to retry>
remote_termination: <bool flag indicating whether process can be terminated via rest api>
```

If you are using basic auth, you will also have a basic auth file that looks like this:

```
<username1>: <password1>
<username2>: <password2>
...
```

# Runtime Considerations

For the terraform infra you are managing from kubernetes jobs, it probably makes the most sense to just run this backend as a sidecar to your main terraform pod in which case secure access to the backend server is not much of a concern.

For terraform work not benefiting from such fine-grained networking segmentation, it probably makes sense to run the backend in a more centralised way in which case the following considerations will pop up.

## Load Balancing

State between requests (the lock really) is persisted in etcd, not in the memory of the backend instance, so you can load balance traffic safely across several instances of the backend.

## Security

To run the backend security, you'll need to use basic auth and a tls certificate/key pair for the server.

Assuming you use an internal certificate, the problem of the server certificate validation will surface.

While you can disable server certificate validation in the terraform backend configuration, we do not recommend this. Instead, you can install the certificate of the CA used to sign your server certificate in the operating system trusted store and terraform should honor it (validated on Ubuntu Linux)

## Sidecar Usage

To more easily run this service in a sidecar in a terraform job, you can enable remote termination via an endpoint.

This will allow the main terraform job to terminate the service via a call to its api when it is done.

The call should be to: `POST /termination`

# Key Storage Convention

Assuming that you pass a state key value of `<key>`:
- The metadata info for the state will be stored in `<key>/info`
- chunk number `Y` of version `X` will be stored in `<key>/state/v<X>/<Y-1>`

On state persistence failure, it is possible that the next version after the current version has populated values from the failure. These will be cleared on the next successful state storage.

When a successful state storage happens, the chunks of the previous version are deleted. This is done as part of a transaction and is guaranteed to happen.

# Legacy Migration Support

To facilitate state migration from the legacy terraform etcd provider with automation, the previous format is supported with the following boolean flags in the configuration:

```
legacy_support:
  read: Whether is should look for a legacy state if the state is not found
  clear: Whether is should look for and clear a legacy state when the state is successfully persisted
  slash_support: Whether is should add a slash to the state key when trying to find the legacy state
```

The last option might be puzzling, until you realise that if the etcd key prefix was `<key>`, the legacy terraform backend would put the state in `<key>default`.

So assuming that you wanted a state with the format `/terraform/backend/...`, with the legacy backend, you would declare the key prefix `/terraform/backend/`, but with this backend, you would declare the key prefix `/terraform/backend` as it will add further slashes for you.

Anyways, you can always start with a plan and look at the logs. The backend will indicate in the logs when the state is read from the legacy location and it will also indicate when it cleared the legacy state after a successful state update.