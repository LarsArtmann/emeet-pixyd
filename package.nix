{
  lib,
  buildGoModule,
  src,
  templ,
}:
let
  version = "0.3.0";
in
buildGoModule {
  pname = "emeet-pixyd";
  inherit version;

  inherit src;

  vendorHash = "sha256-5G5rvtSy9HmI4TUxeXgwIuur0MZmxMDQY9ZXXqShwUY=";
  proxyVendor = true;

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
  };
}
