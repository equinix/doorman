let _pkgs = import <nixpkgs> { };
in { pkgs ? import (_pkgs.fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs-channels";
  # nixos-unstable @2020-07-18
  rev = "5da2d61bd4fc0f3d7829d132dc9c809ab15f8532";
  sha256 = "0sw0kp399v85p2x9ii13p6zfy747rm2mb4axk9zgshg5wsry5inj";
}) { } }:

with pkgs;

mkShell {
  buildInputs = [ dep gitAndTools.git-crypt go golangci-lint protobuf shellcheck shfmt vagrant ]
    ++ pkgs.lib.optional stdenv.isLinux ipset;
}
