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

    users.mutableUsers = false;
  };

  nixos = ((import <nixpkgs/nixos>) {
    configuration = configuration;
    system = "x86_64-linux";
  });

  image = nixos.config.system.build.googleComputeImage;
in 
  pkgs.runCommand "image.tar.gz" {} ''
    cp --reflink=auto ${image}/*.tar.gz $out
  ''