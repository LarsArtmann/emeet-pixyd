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

    port = lib.mkOption {
      type = lib.types.port;
      # Keep in lockstep with pixy.DefaultWebAddr in internal/pixy/pixy.go
      # (the Go default 127.0.0.1:8090) — change both or neither.
      default = 8090;
      description = ''
        TCP port for the web UI and control API. The daemon binds
        127.0.0.1 only — deliberately: the API has no authentication
        and exposes webcam control endpoints (tracking, PTZ, presets,
        privacy), so anything beyond loopback must go through an
        authenticating reverse proxy. This option changes the port,
        never the loopback bind.

        See https://emeet-pixyd.lars.software/guides/monitoring/ for the
        monitoring semantics of the exposed endpoints and the
        reverse-proxy security model.
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

    updateCheck = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = ''
        Let the daemon check GitHub for newer releases (once shortly
        after startup, then daily) and surface a small "update available"
        notice (web panel footer, /api/status, CLI version, Waybar
        tooltip). The check is an unauthenticated GET to
        api.github.com — no version or identifying data is sent. Set to
        false to keep the daemon fully offline.
      '';
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
            EMEET_PIXYD_WEB_ADDR = "127.0.0.1:${toString cfg.port}";
          }
          // lib.optionalAttrs cfg.debug {
            EMEET_PIXYD_DEBUG = "true";
          }
          // lib.optionalAttrs (!cfg.updateCheck) {
            EMEET_PIXYD_UPDATE_CHECK = "off";
          };
        in
        {
          Type = "notify";
          ExecStart = lib.getExe cfg.package;
          Restart = "on-failure";
          RestartSec = 3;
          # The daemon pets the watchdog once per poll cycle (main.go
          # eventLoop ticker -> sdNotify("WATCHDOG=1"); POLL_INTERVAL
          # defaults to 2s). Invariant: WatchdogSec must stay well above
          # the configured poll interval (>= 10x the 2s default) or an
          # operator who raises EMEET_PIXYD_POLL_INTERVAL via an override
          # would get watchdog kills between pings. Keep this in lockstep
          # with the sdNotify site when changing either side.
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
