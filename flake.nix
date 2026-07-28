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
      version = self.rev or self.dirtyRev or "dev";

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
        let
          # go-branded-id v0.5.0 ships a committed compiled binary (`namer`) that
          # embeds nix store paths, which makes the go-modules fixed-output
          # derivation reference the Go toolchain and fail verification. Fetch the
          # same version's source with the binary stripped and wire it in via an
          # in-sandbox `replace` (preBuild) so the poisoned module is never
          # downloaded. Remove goBrandedSrc + replaceBrandedId (and bump go.mod)
          # once go-branded-id publishes a version without the committed binary.
          goBrandedSrc = pkgs.fetchFromGitHub {
            owner = "LarsArtmann";
            repo = "go-branded-id";
            rev = "v0.5.0";
            hash = "sha256-Y7JOypze37axdiU9RiGHwq5dgnIU6PWd/IHPzvjiV48=";
            postFetch = ''
              rm -f "$out/namer"
            '';
          };

          replaceBrandedId = "go mod edit -replace=github.com/larsartmann/go-branded-id@v0.5.0=${goBrandedSrc}";
        in
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
              inherit
                src
                version
                goBrandedSrc
                replaceBrandedId
                ;
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
              vendorHash = "sha256-zawNYoJyvw9fGGBSLlIIltvij6gQ2si0MvJ1OgEEH70=";
              proxyVendor = true;
              doCheck = false;

              nativeBuildInputs = [
                pkgs.templ
                pkgs.golangci-lint
              ];

              GOWORK = "off";
              GOEXPERIMENT = "jsonv2";

              preBuild = ''
                templ generate
                ${replaceBrandedId}
              '';

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
            GOEXPERIMENT = "jsonv2";
          };
        };

      flake = {
        overlays.default = final: _prev: {
          emeet-pixyd = self.packages.${final.system}.emeet-pixyd;
        };

        nixosModules.default = import ./modules/nixos.nix;
      };
    };
}
