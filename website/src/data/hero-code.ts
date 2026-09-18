// Single source of truth for the hero terminal (former TODO #135).
// The copy button derives from heroCode; the displayed HTML derives from
// heroLines - the two can no longer drift apart.

export type HeroTokenKind = 'muted' | 'accent' | 'amber' | 'inline' | 'plain';

export type HeroToken = { text: string; kind: HeroTokenKind };

export type HeroLine = HeroToken[];

const t = (text: string, kind: HeroTokenKind = 'plain'): HeroToken => ({ text, kind });

export const heroLines: HeroLine[] = [
	[t('# Start the daemon', 'muted')],
	[t('emeet-pixyd', 'accent')],
	[],
	[t('# Check status', 'muted')],
	[t('emeet-pixy status')],
	[],
	[t('# Auto mode handles everything', 'muted')],
	[t('# Call starts  -> face tracking + noise cancellation + audio switch', 'inline')],
	[t('# Call ends    -> privacy mode (hardware lens block)', 'inline')],
	[],
	[t('# Manual control also available', 'muted')],
	[t('emeet-pixy '), t('track', 'amber'), t('          '), t('# Enable face tracking', 'inline')],
	[t('emeet-pixy '), t('privacy', 'amber'), t('        '), t('# Privacy mode', 'inline')],
	[t('emeet-pixy '), t('center', 'amber'), t('         '), t('# Center camera', 'inline')],
	[t('emeet-pixy '), t('pan -90', 'amber'), t('        '), t('# Pan left 90 degrees', 'inline')],
	[t('emeet-pixy '), t('zoom 120', 'amber'), t('       '), t('# Zoom to 120%', 'inline')],
];

const kindClass: Record<HeroTokenKind, string> = {
	muted: 'text-text-muted',
	accent: 'text-accent',
	amber: 'text-amber',
	inline: 'text-code-comment',
	plain: '',
};

export function escapeHtml(text: string): string {
	return text.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
}

export function tokenClass(kind: HeroTokenKind): string {
	return kindClass[kind];
}

export function renderLineHtml(line: HeroLine): string {
	return line
		.map((token) => {
			const cls = tokenClass(token.kind);

			return cls ? `<span class="${cls}">${escapeHtml(token.text)}</span>` : escapeHtml(token.text);
		})
		.join('');
}

export function renderHighlightedHtml(lines: HeroLine[]): string {
	return lines.map(renderLineHtml).join('\n');
}

export function renderPlainText(lines: HeroLine[]): string {
	return lines.map((line) => line.map((token) => token.text).join('')).join('\n');
}

export const heroCode = renderPlainText(heroLines);
