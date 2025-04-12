{
  description = "Wedding invite";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.systems.url = "github:nix-systems/default";
  inputs.flake-utils = {
    url = "github:numtide/flake-utils";
    inputs.systems.follows = "systems";
  };
  inputs.templ.url = "github:a-h/templ";

  outputs = { nixpkgs, flake-utils, templ, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        templOverlay = system: templ.packages.${system}.templ;
      in {
        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.libcap
            pkgs.go
            pkgs.gcc
            pkgs.flyctl
            pkgs.golines
            pkgs.air
            pkgs.sqlite
          ];
          buildInputs = [ (templOverlay system) ];
        };
      });
}
