# About

Hashicorp has deprecated their etcd backend.

Even prior to the deprecation, their support for etcd as a backend was not very good as they bundled the entire terraform state in a single etcd key (making it more likely that etcd's 1.5MB key size limit would impact the maximum possible scope of your terraform orchestrations) instead of distributing the state across many keys.

This repo means to remedy that and provide you with the etcd terraform backend that you deserve by abstracting etcd behind an http server (leveraging terraform's http backend).