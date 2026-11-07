{
  description = "EMEET PIXY webcam auto-activation daemon — face tracking, privacy, noise cancellation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
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
      # Inline systems: nixpkgs 26.11 dropped x86_64-darwin, which github:nix-systems/default still lists.
      # Linux-only daemon (package.nix sets lib.platforms.linux): aarch64-darwin
      # would make every eval of packages.aarch64-darwin.* refuse — keeping it
      # listed was why `nix flake check --all-systems` could never pass.
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

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
              templ.enable = true;
              nixfmt.enable = true;
            };
            # Generated files are owned by `templ generate` (devShell and CI
            # regenerate them; the nix derivations consume the committed
            # files — a stale committed _templ.go fail-fasts at nix build
            # instead, PR #19): gofumpt would rewrite
            # generator output (var x = vs x := churn between templ versions)
            # and oscillate against every regenerate. goimports is NOT in
            # treefmt because its go/packages driver shells out to `go`, which
            # wants a toolchain download the offline check sandbox cannot
            # perform; import grouping is enforced by golangci-lint (gci +
            # goimports) in CI and the pre-commit gate instead.
            settings.formatter.gofumpt.excludes = [
              "vendor/*"
              "*_templ.go"
              "**_templ.go"
            ];
            # treefmt-nix's default templ wrapper carries the nixpkgs default
            # `go` (1.26.x): templ fmt parses the generated code via
            # go/packages, and that `go` sees go.mod's `go 1.27` and tries a
            # toolchain download — which the offline check sandbox cannot do.
            # Pin the wrapper's go to go_1_27 so no download is ever needed.
            settings.formatter.templ.command = lib.mkForce (
              pkgs.writeShellApplication {
                name = "templ";
                runtimeInputs = [
                  pkgs.go_1_27
                  pkgs.templ
                ];
                text = ''
                  exec templ "$@"
                '';
              }
            );
          };

          packages = {
            emeet-pixyd = pkgs.callPackage ./package.nix {
              inherit
                src
                version
                ;
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
              vendorHash = "sha256-Lbk/4A571Ix/HB8tuRlhlXYXfSi5ck9zcxjQxJ/BUks=";
              proxyVendor = true;
              doCheck = false;

              nativeBuildInputs = [
                pkgs.golangci-lint
              ];

              GOWORK = "off";

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

              nodes.defaults =
                { pkgs, ... }:
                {
                  imports = [ self.nixosModules.default ];

                  users.users.pixy = {
                    isNormalUser = true;
                    extraGroups = [ "video" ];
                  };

                  # port deliberately UNSET: this node pins the default that
                  # pixy.DefaultWebAddr (internal/pixy/pixy.go) and downstream
                  # port registries (SystemNix's ports.emeet-pixyd) couple to.
                  hardware.emeet-pixy = {
                    enable = true;
                    package = self.packages.${pkgs.system}.emeet-pixyd;
                    user = "pixy";
                  };
                };

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
                    port = 8091;
                  };
                };

              testScript = ''
                defaults.start()

                with subtest("unset port renders the daemon's loopback default"):
                    unit = defaults.succeed(
                        "cat /etc/systemd/user/emeet-pixyd.service"
                    )
                    assert "EMEET_PIXYD_WEB_ADDR=127.0.0.1:8090" in unit, \
                        "default port drifted from the 8090 convention"

                defaults.shutdown()
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
                    # The port option must flow into the rendered unit as a
                    # loopback WEB_ADDR — downstream port registries couple
                    # to this knob, a silent convention match is not enough.
                    assert "EMEET_PIXYD_WEB_ADDR=127.0.0.1:8091" in unit

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
              pkgs.v4l-utils
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
