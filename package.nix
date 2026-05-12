{
  lib,
  buildGoModule,
  src,
  templ,
}:
let
  version = "0.2.0";
in
buildGoModule {
  pname = "emeet-pixyd";
  inherit version;

  inherit src;

  vendorHash = "sha256-F7uprfUn4TCWr7gFi/GbvsfTsf1D/jH2qqgXC3wHTuI=";
  proxyVendor = true;

  nativeBuildInputs = [templ];

  preBuild = ''
    templ generate
  '';

  ldflags = ["-s" "-w" "-X main.buildVersion=${version}"];

  postInstall = ''
    ln -s $out/bin/emeet-pixyd $out/bin/emeet-pixy
  '';

  meta = {
    description = "Auto-activation daemon for EMEET PIXY webcam — face tracking, privacy, noise cancellation";
    homepage = "https://github.com/LarsArtmann/emeet-pixyd";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
    mainProgram = "emeet-pixyd";
  };
}
