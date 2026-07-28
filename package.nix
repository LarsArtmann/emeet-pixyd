{
  lib,
  buildGoModule,
  src,
  version,
  templ,
  goBrandedSrc,
  replaceBrandedId,
}:
buildGoModule {
  pname = "emeet-pixyd";
  inherit version;

  inherit src;

  vendorHash = "sha256-zawNYoJyvw9fGGBSLlIIltvij6gQ2si0MvJ1OgEEH70=";
  proxyVendor = true;

  GOEXPERIMENT = "jsonv2";

  doCheck = false;

  nativeBuildInputs = [ templ ];

  preBuild = ''
    templ generate
    ${replaceBrandedId}
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
