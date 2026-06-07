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

      version = self.shortRev or self.dirtyRev or "dev";

      lib = nixpkgs.lib;

      sourceFiles = lib.fileset.unions [
        (lib.fileset.fileFilter (
          file:
          (lib.hasSuffix ".go" file.name && !lib.hasSuffix "_test.go" file.name)
          || lib.hasSuffix ".mod" file.name
          || lib.hasSuffix ".sum" file.name
          || lib.hasSuffix ".templ" file.name
        ) ./.)
        ./static
      ];

      src = lib.fileset.toSource {
        root = ./.;
        fileset = sourceFiles;
      };

      checkSourceFiles = lib.fileset.unions [
        (lib.fileset.fileFilter (
          file:
          lib.hasSuffix ".go" file.name
          || lib.hasSuffix ".mod" file.name
          || lib.hasSuffix ".sum" file.name
          || lib.hasSuffix ".templ" file.name
        ) ./.)
        ./static
        ./.golangci.yml
      ];

      checkSrc = lib.fileset.toSource {
        root = ./.;
        fileset = checkSourceFiles;
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

          lint = pkgs.buildGoModule {
            pname = "emeet-pixyd-lint";
            inherit version;
            src = checkSrc;
            vendorHash = "sha256-7FOSH+F2YBf0KS7HPLKlOXOEzeArOwaSWN0hJIhWexc=";
            proxyVendor = true;
            doCheck = false;

            nativeBuildInputs = [
              pkgs.templ
              pkgs.golangci-lint
            ];

            GOWORK = "off";

            preBuild = "templ generate";

            buildPhase = ''
              runHook preBuild
              export HOME=$TMPDIR
              golangci-lint run --timeout 2m ./...
              runHook postBuild
            '';

            installPhase = ''
              runHook preInstall
              mkdir -p $out
              runHook postInstall
            '';
          };
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
          meta = {
            mainProgram = "emeet-pixyd";
            description = "EMEET PIXY webcam auto-activation daemon";
          };
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
