variable "google_cloud_project" {}

variable "ssh_pub_key" {
  default = "~/.ssh/id_rsa.pub"
}

provider "google" {
  project = "${var.google_cloud_project}"
  region  = "us-central1"
}

resource "nix_build" "nixpkgs" {
  # Path to the nix expression to build.
  # If expression is set, this is an output path of
  # an expression to write to disk.
  #
  # In this example we make an explicit version of nix packages
  # for other builds to use as their NIX_PATH.
  expression_path = "./nixpkgs.nix"

  # A nix expression to write at expression_path.
  # If expression is not specified, it is assumed
  # the expression is not under terraform control and is
  # written separately.
  #
  # For the curious, the expression is written to disk to ensure relative
  # paths in nix expressions work as intended.
  # expression = ""

  # A nix gc root into the nix store.
  # Same as what you get from nix-build -o ...
  out_link = "./pinned_nixpkgs"
}

resource "nix_build" "nixosimage" {
  # The nix path used to build the expression, if not set, it is taken from the environment.
  nix_path = "nixpkgs=${nix_build.nixpkgs.store_path}:sshpubkey=${pathexpand("${var.ssh_pub_key}")}"

  # We can inline expressions, but be sure to escape them properly.
  # In this example, this expression is building a base vm image to be uploaded to google cloud.
  expression = <<-EOF
  let
    pkgs = ((import <nixpkgs>) {});
    
    configuration = {config, pkgs, ...}: {
      imports = [
        <nixpkgs/nixos/modules/virtualisation/google-compute-image.nix>
      ];

      users.users.root = {
        openssh.authorizedKeys.keys = [
          (builtins.readFile <sshpubkey>)
        ];
      };

    };

    nixos = ((import <nixpkgs/nixos>) {
      configuration = configuration;
      system = "x86_64-linux";
    });

    image = nixos.config.system.build.googleComputeImage;
  in 
    pkgs.runCommand "image.tar.gz" {} ''
      cp --reflink=auto $${image}/*.tar.gz $out
    ''
  EOF

  # Path to the nix expression to possible write, and then build.
  expression_path = "./vmimage-generated.nix"

  # Same as what you get from nix-build -o ...
  out_link = "./nixosimage"
}

resource "random_id" "example_suffix" {
  byte_length = 8
}

resource "google_storage_bucket" "vmimage_bucket" {
  name     = "provider-nix-example-${random_id.example_suffix.hex}"
  location = "US"
}

resource "google_storage_bucket_object" "nixosimage" {
  name   = "nixosimage-${random_id.example_suffix.hex}.raw.tar.gz"
  bucket = "${google_storage_bucket.vmimage_bucket.name}"
  source = "${nix_build.nixosimage.store_path}"
}

resource "google_compute_image" "nixosimage" {
  name = "test-nixos-image-${random_id.example_suffix.hex}"

  raw_disk {
    source = "https://storage.googleapis.com/${google_storage_bucket.vmimage_bucket.name}/${google_storage_bucket_object.nixosimage.name}"
  }

  depends_on = ["google_storage_bucket.vmimage_bucket", "google_storage_bucket_object.nixosimage"]
}

resource "google_compute_instance" "exampleserver" {
  name         = "nixexampleserver-${random_id.example_suffix.hex}"
  machine_type = "f1-micro"
  zone         = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "${google_compute_image.nixosimage.self_link}"
      size  = 20
    }
  }

  network_interface {
    network = "default"

    access_config {}
  }
}

resource "nix_nixos" "nixos" {
  # Used with nixos-rebuild switch --target-host
  target_host = "${google_compute_instance.exampleserver.network_interface.0.access_config.0.nat_ip}"

  # Same as nix_build resource.
  nix_path = "nixpkgs=${nix_build.nixpkgs.store_path}:sshpubkey=${pathexpand("${var.ssh_pub_key}")}"

  # An optional configuration to write to nixos_config_path.
  # If this is not set, the configuration is assumed to already exist.
  # 
  # It is probably best to put most of your config in an existing file, then
  # only write pass some configuration from here.
  nixos_config = <<-EOF
  {config, pkgs, ...}:
  {
    imports = [
      <nixpkgs/nixos/modules/virtualisation/google-compute-image.nix>
    ];

    users.users.root = {
      openssh.authorizedKeys.keys = [
        (builtins.readFile <sshpubkey>)
      ];
    };

    users.motd = ''
      Welcome, here is how you can specify terraform values in a nixos config:
      
      ${random_id.example_suffix.hex}

      You can use this for specifying ip addresses or other terraform provisioned items.
    '';
  }
  EOF

  # Path to your nixos config. If nixos_config is set, this is written, otherwise
  # this file is assumed to exist.
  nixos_config_path = "./configuration-generated.nix"

  # You can run code locally before or after a switch completes.
  # The default is to do nothing, but this shows how you may use it to ssh into the host.
  # The pre/post switch hooks are good places to load secrets or other things you may need to do.
  pre_switch_hook = <<-EOF
  #! /bin/sh
  set -eu
  ssh $NIX_TARGET_USER@$NIX_TARGET_HOST $NIX_SSHOPTS uname -a
  EOF

  # Optional values, with defaults.

  # post_switch_hook = ""

  # Used by nixos-rebuild switch and nixos-rebuild build as --build-host.
  # build_host = "localhost"

  # Time to wait for ssh to become responsive. 
  # ssh_timeout = 180

  # Options passed to ssh when checking or switching your installation.
  # ssh_opts     = "-o StrictHostKeyChecking=accept-new -o BatchMode=yes"

  # Run nix-collect-garbage -d on target host before installing an update.
  # collect_garbage = true

  # SSH commands will run as this user, note they must be able to install the system
  # so values other than root mean little.
  # target_user = "root"
}
