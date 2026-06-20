{
  lib,
  buildGoModule,
  src,
  version,
  templ,
}:
buildGoModule {
  pname = "emeet-pixyd";
  inherit version;

  inherit src;

  # NOTE: vendorHash must be updated after adding cqrs-htmx/v2 as a dependency.
  # cqrs-htmx depends on go-cqrs-lite (github.com/larsartmann/go-cqrs-lite),
  # which is currently a PRIVATE GitHub repository. The nix FOD sandbox cannot
  # fetch private repos, so `nix build` fails at the go-modules derivation.
  #
  # To fix the nix build: either make go-cqrs-lite public, or compute the
  # vendorHash manually by running `go mod download` locally (where
  # GOPRIVATE=github.com/larsartmann/* and SSH keys are configured) and
  # then running `nix hash path $(go env GOMODCACHE)/cache/download`.
  #
  # The old vendorHash (pre-cqrs-htmx) is kept as a placeholder.
  vendorHash = "sha256-V9odnSmOX8+YAKjwhNrSdQn49OzUVGKKCrHfTZNK+9k=";
  proxyVendor = true;

  doCheck = false;

  nativeBuildInputs = [ templ ];

  preBuild = ''
    templ generate
  '';

  ldflags = [
    "-s"
    "-w"
    "-X main.buildVersion=${version}"
  ];

  postInstall = ''
    ln -s $out/bin/emeet-pixyd $out/bin/emeet-pixy
  '';

  meta = {
    description = "Auto-activation daemon for EMEET PIXY webcam — face tracking, privacy, noise cancellation";
    homepage = "https://github.com/LarsArtmann/emeet-pixyd";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
    mainProgram = "emeet-pixyd";
    inherit version;
  };
}
