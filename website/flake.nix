{
  description = "emeet-pixyd website — Astro + Starlight";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    systems.url = "github:nix-systems/default";

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
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        { config, pkgs, ... }:
        let
          mkApp = name: runtimeInputs: text: {
            type = "app";
            program = "${
              pkgs.writeShellApplication {
                inherit name runtimeInputs text;
              }
            }/bin/${name}";
          };
        in
        {
          apps = {
            dev = mkApp "dev" [ pkgs.nodejs ] "npm run dev";
            build = mkApp "build" [ pkgs.nodejs ] "npm run build";
            preview = mkApp "preview" [ pkgs.nodejs ] "npm run preview";
            deploy =
              mkApp "deploy"
                [
                  pkgs.nodejs
                  pkgs.firebase-tools
                ]
                ''
                  npm run build
                  firebase deploy --only hosting
                '';
          };

          devShells.default = pkgs.mkShellNoCC {
            packages = builtins.attrValues {
              inherit (pkgs) nodejs firebase-tools;
            };
          };

          treefmt.programs.nixfmt.enable = true;

          checks.format = config.treefmt.build.check self;
        };
    };
}
