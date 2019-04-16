# Example

This example provisions a vm on google cloud using terraform.
The version of nixos is fixed via the nixpkgs.nix file in this directory.
The configuration.nix file can be updated and terraform apply will update
your server.

After building the plugin in the parent directory you can create the example infrastructure with:

```
terraform init -plugin-dir ../ 
terraform apply -var google_cloud_project=your_project_id
...
terraform destroy -var google_cloud_project=your_project_id
```

## Files

### example.tf

The file terraform loads to kick things off. This file defines all the resources and the dependency relationships between them.

The defined resources include:

- A local pinned version of nixpkgs.
- A local disk image built from the pinned nixpkgs.
- A google cloud bucket to store our uploaded disk image.
- An uploaded and bootable copy of the disk image for google cloud to start servers with.
- A provisioned server based on this image.
- A managed nixos installation that can be updated in place instead of recreating the vm on change.

Terraform does all the work of building and uploading them.

### nixpkgs.nix + update-nixpkgs

A nix expression that downloads the nixpkgs repository (the definition of our operating system and all out software).

The update script downloads the latest published version and saves that. This is how we ensure our builds
are always the same, until we manually update things by running this script.

Our terraform file uses this version of nixpkgs to build all of our resources.

### vmimage.nix

A nix expression that can build a virtual machine image that runs on google cloud. It has the bare minimum 
configuration to allow us to ssh in and apply the config from configuration.nix.

### configuration.nix

The configuration of the deployed server, when this changes and terraform apply is executed,
any changes will be sent to the running server.

### ssh_into_server

A simple script you can run to get an ssh session on your running server, useful so you don't need
to lookup the ip address.