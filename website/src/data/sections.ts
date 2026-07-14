import type { StepCard, ComparisonItem, UseCase, ComparisonMatrix } from "./types";

export const steps: StepCard[] = [
  {
    step: "1",
    stepColor: "accent",
    title: "Detect Call",
    desc: "Polling ticker scans /proc/*/fd every 2s for processes holding the video device open.",
    code: "// /proc scanning + debounce (3 cycles)",
  },
  {
    step: "2",
    stepColor: "accent",
    title: "Activate",
    desc: "Face tracking + noise cancellation enabled via HID. PipeWire source switched to PIXY mic.",
    code: "setTracking(tracking) + setAudio(nc)",
  },
  {
    step: "3",
    stepColor: "amber",
    title: "Call Active",
    desc: "Daemon monitors for call end. Web UI shows live preview and PTZ controls.",
    code: "// camera in use, privacy mode off",
  },
  {
    step: "4",
    stepColor: "amber",
    title: "Privacy Mode",
    desc: "Lens physically blocked via hardware shutter. Audio source restored to default.",
    code: "setTracking(privacy) + setSource(default)",
  },
];

export const comparisons: ComparisonItem[] = [
  {
    variant: "Manual",
    accent: false,
    pros: ["No software to install"],
    cons: [
      "Must remember every call",
      "No privacy mode on forget",
      "No audio switching",
      "Per-app configuration",
    ],
  },
  {
    variant: "Browser extension",
    accent: false,
    pros: ["Works for browser calls"],
    cons: [
      "Only works in browser",
      "No Zoom / Teams / OBS support",
      "Cannot control HID hardware",
      "Cannot switch PipeWire sources",
    ],
  },
  {
    variant: "emeet-pixyd",
    accent: true,
    pros: [
      "Works with ANY app",
      "Hardware privacy mode on call end",
      "Auto audio source switching",
      "Hotplug detection",
      "Full CLI + web UI + Waybar",
    ],
    cons: [],
  },
];

export const comparisonMatrix: ComparisonMatrix = {
  columns: ["Manual", "Browser extension", "emeet-pixyd"],
  rows: [
    { feature: "Works with any app", values: ["no", "no", "yes"] },
    { feature: "Auto face tracking", values: ["no", "partial", "yes"] },
    { feature: "Hardware privacy mode", values: ["no", "no", "yes"] },
    { feature: "Audio source switching", values: ["no", "no", "yes"] },
    { feature: "Zoom / Teams / OBS", values: ["no", "no", "yes"] },
    { feature: "USB hotplug detection", values: ["no", "no", "yes"] },
    { feature: "CLI + Web UI + Waybar", values: ["no", "no", "yes"] },
    { feature: "NixOS module", values: ["no", "no", "yes"] },
  ],
};

export const useCases: UseCase[] = [
  {
    title: "Video Calls",
    desc: "Zoom, Teams, Google Meet, Slack, Discord — auto-tracks your face and blocks the lens when you hang up",
    icon: "video",
  },
  {
    title: "Live Streaming",
    desc: "OBS opens /dev/video* and the daemon activates tracking automatically",
    icon: "stream",
  },
  {
    title: "Privacy First",
    desc: "Forget to disable tracking? The hardware shutter engages the moment the camera is released",
    icon: "shield",
  },
];
