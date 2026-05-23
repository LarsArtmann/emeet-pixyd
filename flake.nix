{
  description = "EMEET PIXY webcam auto-activation daemon — face tracking, privacy, noise cancellation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    {
      self,
      nixpkgs,
    }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      srcFilter =
        path: _type:
        let
          b = baseNameOf path;
        in
        !(
          nixpkgs.lib.hasSuffix "_test.go" b
          || nixpkgs.lib.hasSuffix "_fuzz_test.go" b
          || b == "package.nix"
          || b == "flake.nix"
          || b == "flake.lock"
          || b == "coverage.out"
          || b == "docs"
          || b == ".github"
          || b == "vendor"
        );
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          emeet-pixyd = pkgs.callPackage ./package.nix {
            src = pkgs.lib.cleanSourceWith {
              filter = srcFilter;
              src = ./.;
            };
            inherit (pkgs) templ;
          };
          default = self.packages.${system}.emeet-pixyd;
        }
      );

      checks = forAllSystems (system: {
        build = self.packages.${system}.default;
      });

      overlays.default = _final: prev: {
        emeet-pixyd = prev.callPackage ./package.nix {
          src = prev.lib.cleanSourceWith {
            filter = srcFilter;
            src = self;
          };
          inherit (prev) templ;
        };
      };

      nixosModules.default = import ./modules/nixos.nix;

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/emeet-pixyd";
        };
      });

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go_1_26
              golangci-lint
              templ
            ];

            GOWORK = "off";
          };
        }
      );

      formatter = forAllSystems (system: nixpkgs.legacyPackages.${system}.nixfmt);
    };
}
