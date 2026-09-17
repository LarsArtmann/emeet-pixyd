{
  lib,
  buildGoModule,
  src,
  version,
  templ,
  go_1_27,
  ...
}:
buildGoModule {
  pname = "emeet-pixyd";
  inherit version;

  inherit src;

  go = go_1_27;

  vendorHash = "sha256-Her301HadgDLXpZld8pX9VS1w8S2xsDpBtYr07oJfOM=";
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
