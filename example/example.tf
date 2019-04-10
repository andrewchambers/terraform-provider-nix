variable "google_cloud_project" {}

variable "ssh_pub_key" {
  default = "~/.ssh/id_rsa.pub"
}

provider "google" {
  project = "${var.google_cloud_project}"
  region  = "us-central1"
}

data "nix_build" "nixpkgs" {
  # Path to the nix expression to build.
  # Here we make an explicit version of nix packages
  # for other builds to use.
  expression_path = "./nixpkgs.nix"
}

data "nix_build" "nixosimage" {
  # The nix path used to build the expression, if not set, it is taken from the environment.
  nix_path = "nixpkgs=${data.nix_build.nixpkgs.store_path}:sshpubkey=${pathexpand("${var.ssh_pub_key}")}"

  # Path to the nix expression to build.
  expression_path = "./vmimage.nix"
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
  source = "${data.nix_build.nixosimage.store_path}"
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

  # Same as data source above.
  nix_path = "nixpkgs=${data.nix_build.nixpkgs.store_path}:sshpubkey=${pathexpand("${var.ssh_pub_key}")}"

  # Path to your nixos config.
  nixos_config = "./configuration.nix"

  # You can run code locally before or after a switch completes.
  # The default is to do nothing, but this shows how you may use it to ssh into the host.
  # The pre/post switch hooks are good places to load secrets or other things you may need to do.
  #
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
  # Note 'accept-new' requires a 'newish' openssh.
  # ssh_opts     = "-o StrictHostKeyChecking=accept-new -o BatchMode=yes"

  # Run nix-collect-garbage -d on target host before installing an update.
  # collect_garbage = true

  # SSH commands will run as this user, note they must be able to install the system,
  # so values other than root mean little.
  # target_user = "root"
}
