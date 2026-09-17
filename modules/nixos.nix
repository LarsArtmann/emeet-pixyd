{
  pkgs,
  lib,
  config,
  ...
}:
let
  cfg = config.hardware.emeet-pixy;
in
{
  options.hardware.emeet-pixy = {
    enable = lib.mkEnableOption "EMEET PIXY webcam auto-activation daemon";

    package = lib.mkPackageOption pkgs "emeet-pixyd" { };

    user = lib.mkOption {
      type = lib.types.str;
      description = ''
        User that owns the runtime state directory (/run/emeet-pixyd).
        Must match the user running the graphical session, since the
        daemon runs as a systemd user service under that session.
      '';
    };

    auto = lib.mkOption {
      type = lib.types.enum [
        "off"
        "full"
        "tracking-only"
        "privacy-only"
      ];
      default = "full";
      description = ''
        Automatic camera management strategy:
          off            — manual control: no /proc monitoring, camera keeps whatever mode you set
                            (re-applied automatically after reboots, replugs, and power cycles)
          full           — tracking + noise cancellation + PipeWire source on call start, privacy on call end
          tracking-only  — face tracking on call start, privacy on call end (no audio/source switching)
          privacy-only   — privacy mode on call end (no call-start activation)

        See https://emeet-pixyd.lars.software/guides/auto-modes/ for the full reference.
      '';
    };

    defaultAudio = lib.mkOption {
      type = lib.types.enum [
        "nc"
        "live"
        "org"
      ];
      default = "nc";
      description = "Default audio mode (nc=noise cancel, live, org=original)";
    };

    debug = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Enable debug mode (pprof endpoints at /debug/pprof/)";
    };
  };

  config = lib.mkIf cfg.enable {
    services.udev.extraRules = ''
      # EMEET PIXY HID access for camera control (tracking, audio, gesture, privacy).
      # 00c0 = PIXY, 0118 = PIXY 2K (issue #6).
      KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="328f", ATTRS{idProduct}=="00c0|0118", GROUP="video", MODE="0660", TAG+="uaccess"
      # EMEET PIXY video device access
      SUBSYSTEM=="video4linux", ATTRS{idVendor}=="328f", ATTRS{idProduct}=="00c0|0118", GROUP="video", MODE="0660", TAG+="uaccess"
    '';

    systemd.tmpfiles.rules = [
      "d /run/emeet-pixyd 0755 ${cfg.user} video -"
    ];

    systemd.user.services.emeet-pixyd = {
      description = "EMEET PIXY Webcam Auto-Activation Daemon";
      after = [
        "pipewire.service"
        "graphical-session.target"
      ];
      wants = [ "pipewire.service" ];
      partOf = [ "graphical-session.target" ];
      wantedBy = [ "graphical-session.target" ];

      serviceConfig =
        let
          envVars = {
            EMEET_PIXYD_AUTO = cfg.auto;
            EMEET_PIXYD_DEFAULT_AUDIO = cfg.defaultAudio;
          }
          // lib.optionalAttrs cfg.debug {
            EMEET_PIXYD_DEBUG = "true";
          };
        in
        {
          Type = "notify";
          ExecStart = lib.getExe cfg.package;
          Restart = "on-failure";
          RestartSec = 3;
          WatchdogSec = "30";
          OOMScoreAdjust = -100;

          ProtectSystem = "strict";
          ReadWritePaths = [ "/run/emeet-pixyd" ];
          PrivateTmp = true;
          NoNewPrivileges = true;
          RestrictAddressFamilies = [
            "AF_UNIX"
            "AF_NETLINK"
            "AF_INET"
          ];
          MemoryMax = "256M";

          Environment = lib.concatStringsSep " " (lib.mapAttrsToList (k: v: "${k}=${v}") envVars);
        };

      path = [
        pkgs.v4l-utils
        pkgs.wireplumber
        pkgs.libnotify
        pkgs.ffmpeg-headless
      ];
    };
  };
}
