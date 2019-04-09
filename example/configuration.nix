{config, pkgs, ...}:
{
  imports = [
    <nixpkgs/nixos/modules/virtualisation/google-compute-image.nix>
  ];

  users.users.root = {
    openssh.authorizedKeys.keys = [
      (builtins.readFile /home/ac/.ssh/id_rsa.pub)
    ];
  };

  users.mutableUsers = false;
}
