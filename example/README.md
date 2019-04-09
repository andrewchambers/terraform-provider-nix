# Example

This example provisions a vm on google cloud using terraform.
The version of nixos is fixed via the nixpkgs.nix file in this directory.
The configuration.nix file can be updated and terraform apply will update
your server.

After building the plugin in the parent directory, you can create the example infrastructure with:

```
terraform init -plugin-dir ../ 
terraform apply -var google_cloud_project=your_project_id
```

Enjoy.