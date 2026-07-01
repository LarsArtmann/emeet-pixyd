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

  vendorHash = "sha256-v+Btv34kWoWz0gONlTjVKR7c2JklY3zOl5U9oglb9MY=";
  proxyVendor = true;

  doCheck = false;

  nativeBuildInputs = [ templ ];

  preBuild = ''
    templ generate
    # Validate generated files are non-empty (guards against intermittent
    # zero-byte *_templ.go output that breaks the build silently).
    for f in *_templ.go; do
      if [ ! -s "$f" ]; then
        echo "WARNING: $f is empty after templ generate — re-running" >&2
        templ generate
        if [ ! -s "$f" ]; then
          echo "FATAL: $f still empty after retry" >&2
          exit 1
        fi
      fi
    done
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
