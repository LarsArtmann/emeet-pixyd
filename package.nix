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

  vendorHash = "sha256-LdB/PtHu4QJH7y2QLHxs5zHuvcgJpW4KT9W9Rf4324Q=";
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
