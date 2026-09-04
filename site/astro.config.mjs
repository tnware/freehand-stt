// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

const productionUrl = new URL(
	process.env.SITE_URL ?? 'https://tnware.github.io/freehand-stt/',
);
const basePath = process.env.CI ? productionUrl.pathname : '/';
const base = basePath.replace(/\/$/, '');

// https://astro.build/config
export default defineConfig({
	site: productionUrl.origin,
	base: basePath,
	redirects: {
		'/docs': `${base}/docs/getting-started`,
	},
	integrations: [
		starlight({
			title: 'Freehand',
			description:
				'A native desktop client for self-hosted and OpenAI-compatible speech infrastructure.',
			// Starlight applies Astro's configured base path to root-relative
			// assets. Supplying basePath here would double-prefix GitHub Pages.
			favicon: '/favicon.svg',
			logo: {
				src: './src/assets/freehand-mark.svg',
				alt: 'Freehand',
			},
			social: [
				{
					icon: 'github',
					label: 'Freehand on GitHub',
					href: 'https://github.com/tnware/freehand-stt',
				},
			],
			editLink: {
				baseUrl: 'https://github.com/tnware/freehand-stt/edit/main/site/',
			},
			customCss: ['./src/styles/docs.css'],
			lastUpdated: true,
			sidebar: [
				{ label: 'Freehand home', link: '/' },
				{ label: 'Download Freehand', link: '/download/' },
				{
					label: 'Get started',
					items: [
						{ slug: 'docs/getting-started' },
						{ slug: 'docs/guides/windows-installer' },
						{ slug: 'docs/guides/connect-a-server' },
					],
				},
				{
					label: 'Use Freehand',
					items: [
						{ slug: 'docs/guides/using-freehand' },
						{ slug: 'docs/guides/post-processing' },
						{ slug: 'docs/guides/privacy-and-safety' },
						{ slug: 'docs/guides/troubleshooting' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ slug: 'docs/reference/protocol' },
						{ slug: 'docs/reference/shortcuts' },
					],
				},
				{
					label: 'Contribute',
					items: [
						{ slug: 'docs/development' },
					],
				},
			],
			head: [
				{
					tag: 'meta',
					attrs: { name: 'theme-color', content: '#071120' },
				},
			],
		}),
	],
});
