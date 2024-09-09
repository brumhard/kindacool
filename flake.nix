{
  description = "local development setup";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    utils,
  }:
    utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {inherit system;};
      in {
        devShell = pkgs.mkShell {
          packages = with pkgs; [
            go_1_22
            (pkgs.writeShellScriptBin "pulumi" ''
              exec ${pkgs.pulumi-bin}/bin/pulumi "$@"
            '')
            earthly
          ];

          shellHook = "
          source .env.sh
          ";
        };
      }
    );
}
