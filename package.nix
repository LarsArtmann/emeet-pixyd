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

  vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
  proxyVendor = true;

  doCheck = false;

  # go-cqrs-lite subdirectory modules need the Go module proxy (not direct VCS)
  # which is blocked in the nix sandbox. The FOD inherits this env var.
  overrideModAttrs = _: {
    GOPROXY = "https://proxy.golang.org,direct";
  };

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
