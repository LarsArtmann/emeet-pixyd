import { defineConfig, fontProviders } from "astro/config";
import starlight from "@astrojs/starlight";
import sitemap from "@astrojs/sitemap";

import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
	site: "https://emeet-pixyd.lars.software",

	security: {
		csp: {
			scriptDirective: {
				resources: ["'self'"],
			},
			styleDirective: {
				resources: ["'self'", "'unsafe-inline'"],
			},
		},
	},

	compressHTML: true,

	prefetch: {
		prefetchAll: false,
		defaultStrategy: "hover",
	},

	fonts: [
		{
			provider: fontProviders.google(),
			name: "Space Grotesk",
			cssVariable: "--font-space-grotesk",
			weights: [300, 400, 500, 600, 700],
			styles: ["normal"],
			subsets: ["latin"],
			fallbacks: ["sans-serif"],
		},
		{
			provider: fontProviders.fontsource(),
			name: "JetBrains Mono",
			cssVariable: "--font-jetbrains-mono",
			weights: [400, 500, 600, 700],
			styles: ["normal"],
			subsets: ["latin"],
			fallbacks: ["monospace"],
		},
	],

	integrations: [
		sitemap(),
		starlight({
			title: "emeet-pixyd",
			favicon: "/favicon.svg",
			customCss: ["./src/styles/starlight.css"],
			lastUpdated: true,
			editLink: {
				baseUrl: "https://github.com/LarsArtmann/emeet-pixyd/edit/master/website",
			},
			expressiveCode: {
				themes: ["github-light", "github-dark"],
				frames: {
					showCopyToClipboardButton: true,
				},
			},
			sidebar: [
				{
					label: "Getting Started",
					items: [
						{ label: "Installation", slug: "getting-started/installation" },
						{ label: "Quick Start", slug: "getting-started/quick-start" },
					],
				},
				{
					label: "Guides",
					items: [
						{ label: "Auto Modes", slug: "guides/auto-modes" },
						{ label: "Web UI", slug: "guides/web-ui" },
						{ label: "CLI Reference", slug: "guides/cli-reference" },
						{ label: "Configuration", slug: "guides/configuration" },
						{ label: "PTZ Control", slug: "guides/ptz-control" },
						{ label: "Presets", slug: "guides/presets" },
						{ label: "Waybar Integration", slug: "guides/waybar" },
						{ label: "Prometheus Metrics", slug: "guides/metrics" },
					],
				},
				{
					label: "Architecture",
					items: [
						{ label: "Overview", slug: "architecture/overview" },
						{ label: "HID Protocol", slug: "architecture/hid-protocol" },
						{ label: "Call Detection", slug: "architecture/call-detection" },
					],
				},
				{
					label: "Community",
					items: [
						{ label: "Troubleshooting", slug: "troubleshooting" },
						{ label: "Changelog", slug: "changelog" },
						{ label: "Contributing", slug: "contributing" },
						{ label: "Related Tools", slug: "related-tools" },
					],
				},
			],
			social: [
				{
					icon: "github",
					label: "GitHub",
					href: "https://github.com/LarsArtmann/emeet-pixyd",
				},
			],
			head: [
				{
					tag: "meta",
					attrs: {
						name: "description",
						content:
							"Auto-activation daemon for the EMEET PIXY dual-camera AI webcam on Linux. Call detection, face tracking, privacy mode, audio switching, and an HTMX web UI.",
					},
				},
			],
		}),
	],

	vite: {
		plugins: [tailwindcss()],
	},
});
