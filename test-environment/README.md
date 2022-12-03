# Overview

To test the backend locally, you need to:

1. Start an etcd server locally
2. Provision tls certifcates for the http backend
3. Run the http backend
4. Run some terraform scripts against the http backend to validate

# Requirements

Step 1 assumes kubernetes running directly on your local machine (ie, exposed nodeports will be accessible from your localhost address).

The script to install the CA on your system in step 2 assumes you are using Debian or a derivative that handles CA installation the same way (validated on Ubuntu here).

Step 3 assumes that you have golang 1.18 or later installed on your system.

# Step 1: Starting Etcd

From the **etcd-server** directory, run the following to launch etcd with mTLS in kubernetes:

```
terraform init
terraform apply
```

You can customize the kubernetes resource prefix, nodeport or kubernetes configuration to use by creating a **local.tfvars** file and running terraform apply this way: `terraform apply -var-file=local.tfvars`

# Step 2: Provision backend tls certificates

From the **localhost-certs** directory, run the following to create a server certificate and accompanying CA:

```
terraform init
terraform apply
```

Until this gets merged, the terraform http backend client won't support specifying a trusted CA to authentify the server so you'll need to install the server's CA certificat in your system's CA store: https://github.com/hashicorp/terraform/pull/31699

For Debian and derivatives, the following command will allow you to install the CA:

```
sudo ./install_cert.sh
```

# Step 3: Start the Backend Server

In the **backend-server** directory, run:

```
./run.sh
```

This will monopolize the current shell prompt, so you'll need to run step 4 in another shell prompt.

# Step 4: Run Local Terraform Scripts Using the Backend

From the **terraform-consumer** directory, you can apply and destroy terraform resources to try out the backend.

The current configuration are set to create and destroy 20 files.