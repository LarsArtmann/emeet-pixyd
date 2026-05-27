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

  vendorHash = "sha256-fq1IgZbTx+KQCKO6OGbM1jBe7NW5AYBWXHgY6FYrD1I=";
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
  };
}
