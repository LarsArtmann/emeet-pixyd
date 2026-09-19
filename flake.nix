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

          packages = {
            emeet-pixyd = pkgs.callPackage ./package.nix {
              inherit
                src
                version
                ;
              inherit (pkgs) templ;
              buildGoModule = pkgs.buildGoModule.override { go = pkgs.go_1_27; };
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
            format = config.treefmt.build.check self;
            build = config.packages.default;

            lint = (pkgs.buildGoModule.override { go = pkgs.go_1_27; }) {
              pname = "emeet-pixyd-lint";
              inherit version;
              src = checkSrc;
              vendorHash = "sha256-nZOWL+rNCy360BQ/NobyyIA4mmsKYnGqf+xmRecq/C0=";
              proxyVendor = true;
              doCheck = false;

              nativeBuildInputs = [
                pkgs.templ
                pkgs.golangci-lint
              ];

              GOWORK = "off";

              preBuild = ''
                templ generate
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

            # Boots a VM with the NixOS module enabled and asserts the
            # module's rendered system state (udev rules for BOTH PIXY
            # product IDs, tmpfiles entry, hardened user unit). The daemon
            # itself cannot run headless (no camera, no graphical session).
            vmTest = pkgs.testers.nixosTest {
              name = "emeet-pixy-module";

              nodes.machine =
                { pkgs, ... }:
                {
                  imports = [ self.nixosModules.default ];

                  users.users.pixy = {
                    isNormalUser = true;
                    extraGroups = [ "video" ];
                  };

                  hardware.emeet-pixy = {
                    enable = true;
                    package = self.packages.${pkgs.system}.emeet-pixyd;
                    user = "pixy";
                    auto = "off";
                  };
                };

              testScript = ''
                machine.start()

                with subtest("udev rules cover both PIXY product IDs"):
                    rules = machine.succeed("cat /etc/udev/rules.d/*.rules")
                    assert 'ATTRS{idProduct}=="00c0|0118"' in rules, \
                        "udev rules missing the 00c0|0118 alternation (issue #6)"
                    assert rules.count('ATTRS{idProduct}=="00c0|0118"') == 2, \
                        "expected one hidraw and one video4linux rule"

                with subtest("state dir tmpfiles rule present"):
                    machine.succeed("grep -q '^d /run/emeet-pixyd' /etc/tmpfiles.d/*.conf")

                with subtest("user service unit rendered with config and hardening"):
                    # cat the canonical path directly. The earlier
                    # `cat $(find /etc/systemd/user ...)` hung forever: NixOS
                    # links /etc/systemd/user into the store as a SYMLINK,
                    # find does not descend it, so the substitution was empty
                    # and bare `cat` blocked reading stdin.
                    unit = machine.succeed(
                        "cat /etc/systemd/user/emeet-pixyd.service"
                    )
                    assert unit.strip() != "", \
                        "emeet-pixyd user unit is empty"
                    assert "ProtectSystem=strict" in unit
                    assert "EMEET_PIXYD_AUTO=off" in unit
                    assert "EMEET_PIXYD_DEFAULT_AUDIO=nc" in unit

                machine.shutdown()
              '';
            };
          };

          devShells.default = pkgs.mkShellNoCC {
            packages = [
              pkgs.go_1_27
              pkgs.golangci-lint
              pkgs.templ
              pkgs.dprint
              pkgs.git
            ];

            GOWORK = "off";

            shellHook = ''
              # Install the pre-commit lint gate (idempotent, store-linked so
              # it always tracks the committed script). Works with both the
              # default .git/hooks and a configured core.hooksPath.
              hook_path="$(git rev-parse --git-path hooks/pre-commit)"
              mkdir -p "$(dirname "$hook_path")"
              ln -sfn "${./scripts/pre-commit}" "$hook_path"
            '';
          };
        };

      flake = {
        overlays.default = final: _prev: {
          emeet-pixyd = self.packages.${final.stdenv.hostPlatform.system}.emeet-pixyd;
        };

        nixosModules.default = import ./modules/nixos.nix;
      };
    };
}
