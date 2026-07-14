import type { Feature } from "./types";

export const features: Feature[] = [
  {
    icon: "eye",
    title: "Call Detection",
    desc: "Scans /proc for processes holding the video device open. Works with Zoom, Teams, Meet, OBS, anything that opens /dev/video*.",
  },
  {
    icon: "bolt",
    title: "Auto Face Tracking",
    desc: "Enables the PIXY's dual-camera AI tracking the moment your call starts. No per-app setup. No manual toggles.",
  },
  {
    icon: "shield",
    title: "Privacy Mode",
    desc: "When the call ends, the lens is physically blocked. The hardware shutter engages automatically.",
  },
  {
    icon: "volume",
    title: "Audio Switching",
    desc: "Auto-switches your PipeWire default source to the PIXY microphone on call start. Noise cancellation included.",
  },
  {
    icon: "globe",
    title: "HTMX Web UI",
    desc: "Dark-themed control panel at 127.0.0.1:8090 with live MJPEG preview, PTZ sliders, presets, and toast notifications.",
  },
  {
    icon: "chart",
    title: "Prometheus Metrics",
    desc: "OpenTelemetry-based metrics exposed at /metrics. Monitor probe results, stream duration, and command latency.",
  },
  {
    icon: "plug",
    title: "USB Hotplug",
    desc: "Netlink uevent listener detects USB plug/unplug in real time. The daemon auto-re-probes without a restart.",
  },
  {
    icon: "terminal",
    title: "Waybar + CLI",
    desc: "JSON output for Waybar modules, full CLI control via Unix socket, and NixOS module with one-option enable.",
  },
];
