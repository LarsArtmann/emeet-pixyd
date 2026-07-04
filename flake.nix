{
  description = "EMEET PIXY webcam auto-activation daemon — face tracking, privacy, noise cancellation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    systems.url = "github:nix-systems/default";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      self,
      nixpkgs,
      flake-parts,
      systems,
      treefmt-nix,
    }:
    let
      version = "0.3.1";

      inherit (nixpkgs) lib;

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
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import systems;

      imports = [
        treefmt-nix.flakeModule
      ];

      perSystem =
        {
          config,
          pkgs,
          ...
        }:
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              gofumpt.enable = true;
              goimports.enable = true;
              templ.enable = true;
              nixfmt.enable = true;
            };
          };

          checks.format = config.treefmt.build.check self;
          packages = {
            emeet-pixyd = pkgs.callPackage ./package.nix {
              inherit src version;
              inherit (pkgs) templ;
            };
            default = config.packages.emeet-pixyd;
          };

          apps.default = {
            type = "app";
            program = "${config.packages.default}/bin/emeet-pixyd";
            meta = {
              mainProgram = "emeet-pixyd";
              description = "EMEET PIXY webcam auto-activation daemon";
              homepage = "https://github.com/LarsArtmann/emeet-pixyd";
              license = {
                shortName = "MIT";
                fullName = "MIT License";
                url = "https://opensource.org/licenses/MIT";
              };
              maintainers = [ ];
              platforms = [
                "x86_64-linux"
                "aarch64-linux"
              ];
            };
          };

          checks = {
            build = config.packages.default;

            lint = pkgs.buildGoModule {
              pname = "emeet-pixyd-lint";
              inherit version;
              src = checkSrc;
              vendorHash = "sha256-C0MAslIennLWlK/Ed2Ua57mqmmTP3f6jhbJ0XRuAvAA=";
              proxyVendor = true;
              doCheck = false;

              nativeBuildInputs = [
                pkgs.templ
                pkgs.golangci-lint
              ];

              GOWORK = "off";

              preBuild = "templ generate";

              buildPhase = ''
                export HOME=$TMPDIR
                export GOCACHE=$TMPDIR/go-cache
                runHook preBuild
                golangci-lint run --timeout 2m ./...
                runHook postBuild
              '';

              installPhase = ''
                runHook preInstall
                mkdir -p $out
                runHook postInstall
              '';
            };
            test = config.packages.default.overrideAttrs (_: {
              doCheck = true;
            });
          };

          devShells.default = pkgs.mkShellNoCC {
            packages = [
              pkgs.go_1_26
              pkgs.golangci-lint
              pkgs.templ
            ];

            GOWORK = "off";
          };
        };

      flake = {
        overlays.default = final: _prev: {
          emeet-pixyd = final.callPackage ./package.nix {
            inherit src version;
            inherit (final) templ;
          };
        };

        nixosModules.default = import ./modules/nixos.nix;
      };
    };
}
