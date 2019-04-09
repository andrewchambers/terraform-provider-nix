let 
  nixpkgsStr = builtins.fetchTarball {url = "https://github.com/NixOS/nixpkgs/archive/91fa6990b2505fb6c01850f13954917f1c168383.tar.gz"; sha256 = "1xsaz9n41p8yxqxf78lh74bbpvgnymdmq1hvnagra7r6bp3jp7ad";};
  pkgs = (import nixpkgsStr) {};
in
  pkgs.runCommand "nixpkgs" {} ''
    ln -sv ${nixpkgsStr} $out
  ''
