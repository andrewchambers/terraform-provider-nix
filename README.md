# terraform-provider-nix

A terraform provider for [nix](https://nixos.org/nix/) builds and [nixos](https://nixos.org/) installations.

With this provider you can do things like:

- Build a vm image from a nix expression and deploy it with terraform.
- Build a docker image from a nix expression and deploy it with terraform.
- Manage a nixos installation via ssh.

Note, this is a completely different project from [the existing plugin in nixpkgs](https://github.com/tweag/terraform-provider-nixos).
Please consult that project to decide which suites your needs better.

## Quickstart

This will run the example which is a good starting point, demonstrating all options. The example
uses google cloud, though I would appreciate contributed examples for other platforms. After running
the example you will have a managed server and vm images.

```
go build
cd example
terraform init -plugin-dir ../
terraform apply -var google_cloud_project=your_project_id
```

## Debugging

To view commands being run, set the env variable TF_LOG=debug.

## Example configuration and options

See the example directory for an example configuration with all valid options specified.

## Status

Working, but want feedback and users.

## Sponsor Messages

This project was sponsored by [backupbox.io](https://backupbox.io)

... *Your message here*

## Sponsoring

This project took time and effort to make, please sponsor the project
via this [paypal donation link](https://www.paypal.com/cgi-bin/webscr?cmd=_s-xclick&hosted_button_id=LX5MPQ26BSWS6&source=url).

Add a markdown message shorter than 70 characters total to your donation it will be added
to the sponsor section. Note that sponsor messages may be rejected at the project
authors judgement.

If you can't afford to sponsor, please consider giving this project a star and share with your
friends.

## Authors

Andrew Chambers - ac@acha.ninja