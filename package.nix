{
  lib,
  buildGoModule,
  src,
  templ,
}:
buildGoModule {
  pname = "emeet-pixyd";
  version = "0.2.0";

  inherit src;

  vendorHash = "sha256-FnVn8EWpWeu/ELZzj4/079qZxftYSOXmF8mwKdS+9KI=";
  proxyVendor = true;

  nativeBuildInputs = [templ];

  preBuild = ''
    templ generate
  '';

  ldflags = ["-s" "-w"];

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
