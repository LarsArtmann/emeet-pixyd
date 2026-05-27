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

      version = self.ref or self.dirtyRef or "dev";

      sourceFiles = nixpkgs.lib.fileset.unions [
        ./go.mod
        ./go.sum
        ./main.go
        ./auto.go
        ./cache.go
        ./commands.go
        ./errors.go
        ./handlers.go
        ./hid.go
        ./main.go
        ./metrics.go
        ./middleware.go
        ./probe.go
        ./process.go
        ./state.go
        ./stream.go
        ./templates.templ
        ./uevent.go
        ./uevent_linux.go
        ./v4l2.go
        ./web_types.go
        ./internal
        ./static
      ];

      src = nixpkgs.lib.fileset.toSource {
        root = ./.;
        fileset = sourceFiles;
      };
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          emeet-pixyd = pkgs.callPackage ./package.nix {
            inherit src version;
            templ = pkgs.templ;
          };
          default = self.packages.${system}.emeet-pixyd;
        }
      );

      checks = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          build = self.packages.${system}.default;
        }
      );

      overlays.default = _final: prev: {
        emeet-pixyd = prev.callPackage ./package.nix {
          inherit src version;
          templ = prev.templ;
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
          default = pkgs.mkShellNoCC {
            packages = [
              pkgs.go_1_26
              pkgs.golangci-lint
              pkgs.templ
            ];

            GOWORK = "off";
          };
        }
      );

      formatter = forAllSystems (system: nixpkgs.legacyPackages.${system}.nixfmt);
    };
}
