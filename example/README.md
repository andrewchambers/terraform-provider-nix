# Example

This example provisions a vm on google cloud using terraform.
The version of nixos is fixed via the nixpkgs.nix file in this directory.
The configuration.nix file can be updated and terraform apply will update
your server.

The example uses the ssh option ```-o StrictHostKeyChecking=accept-new```
which requires openssh 7.5 or newer.

Enjoy :)