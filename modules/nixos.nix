{
  pkgs,
  lib,
  config,
  ...
}: let
  cfg = config.hardware.emeet-pixy;
in {
  options.hardware.emeet-pixy = {
    enable = lib.mkEnableOption "EMEET PIXY webcam auto-activation daemon";

    user = lib.mkOption {
      type = lib.types.str;
      default = "lars";
      description = "User account for the daemon systemd service";
    };

    autoTracking = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable auto face tracking when video call detected";
    };

    autoPrivacy = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Enable privacy mode when no call is active";
    };

    defaultAudio = lib.mkOption {
      type = lib.types.enum ["nc" "live" "org"];
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
    environment.systemPackages = with pkgs; [
      v4l-utils
    ];

    services.udev.extraRules = ''
      # EMEET PIXY HID access for camera control (tracking, audio, gesture, privacy)
      KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="328f", ATTRS{idProduct}=="00c0", GROUP="video", MODE="0660", TAG+="uaccess"
      # EMEET PIXY video device access
      SUBSYSTEM=="video4linux", ATTRS{idVendor}=="328f", ATTRS{idProduct}=="00c0", GROUP="video", MODE="0660", TAG+="uaccess"
    '';

    systemd.tmpfiles.rules = [
      "d /run/emeet-pixyd 0755 ${cfg.user} video -"
    ];

    systemd.user.services.emeet-pixyd = {
      description = "EMEET PIXY Webcam Auto-Activation Daemon";
      after = ["pipewire.service" "graphical-session.target"];
      wants = ["pipewire.service"];
      partOf = ["graphical-session.target"];
      wantedBy = ["graphical-session.target"];

      serviceConfig =
        let
          envVars = {
            EMEET_PIXYD_AUTO =
              if cfg.autoTracking && cfg.autoPrivacy
              then "true"
              else "false";
            EMEET_PIXYD_DEFAULT_AUDIO = cfg.defaultAudio;
          }
          // lib.optionalAttrs cfg.debug {
            EMEET_PIXYD_DEBUG = "true";
          };
        in
        {
          Type = "simple";
          ExecStart = "${pkgs.emeet-pixyd}/bin/emeet-pixyd";
          Restart = "on-failure";
          RestartSec = "3";
          Environment = lib.concatStringsSep " " (
            lib.mapAttrsToList (k: v: "${k}=${v}") envVars
          );
        };

      path = [pkgs.v4l-utils pkgs.wireplumber pkgs.libnotify];
    };
  };
}
