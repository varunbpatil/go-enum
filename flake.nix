{
  description = "go-enum - Go enum code generator";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      allSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        inherit system;
        pkgs = import nixpkgs { inherit system; };
      });
    in
    {
      packages = forAllSystems ({ pkgs, ... }:
        rec {
          default = go-enum;

          go-enum = pkgs.buildGoModule {
            pname = "go-enum";
            version = "0.1.0";
            src = ./.;

            vendorHash = null;

            subPackages = [ "." ];

            env.CGO_ENABLED = 0;

            ldflags = [
              "-s"
              "-w"
            ];
          };
        });

      overlays.default = final: prev: {
        go-enum = self.packages.${final.stdenv.system}.go-enum;
      };

      devShells = forAllSystems ({ pkgs, system }: {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            golangci-lint
          ];
        };
      });
    };
}
