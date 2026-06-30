{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      overlays.default = final: prev: {
        app = prev.callPackage ./package.nix { };
      };
      flake = flake-utils.lib.eachDefaultSystem (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
            overlays = [ self.overlays.default ];
          };
        in
        {
          packages = with pkgs; {
            inherit app;
            default = app;
            dockerImage = callPackage ./docker.nix { inherit app; };
          };
          devShells.default =
            with pkgs;
            mkShell {
              packages = [
                go
                golangci-lint
                gopls
                govulncheck
                just
              ];
            };

          formatter = pkgs.nixfmt-tree;
        }
      );
    in
    flake // { inherit overlays; };
}
