// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import { productionUrl } from './site-url.mjs';
const basePath = process.env.CI ? productionUrl.pathname : '/';
const base = basePath.replace(/\/$/, '');

// https://astro.build/config
export default defineConfig({
	site: productionUrl.origin,
	base: basePath,
	redirects: {
		'/docs': `${base}/docs/getting-started`,
        '/docs/backends/qwen3-asr': `${base}/docs/models/qwen3-asr/`,
        '/docs/guides/model-profiles': `${base}/docs/models/`,
	},
	integrations: [
		starlight({
			title: 'Freehand',
			description:
				'A lightweight Windows client for speech-to-text and text-to-speech services you choose.',
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
            components: { PageTitle: './src/components/DocsPageTitle.astro' },
			lastUpdated: true,
			sidebar: [
				{ label: 'Freehand home', link: '/' },
				{ label: 'Download Freehand', link: '/download/' },
                { label: 'Compare backends', link: '/backends/' },
                { label: 'Explore models', link: '/models/' },
                { label: 'Backend profiles', items: [
                  { slug: 'docs/backends' },
                  { slug: 'docs/backends/generic' },
                  { slug: 'docs/backends/speaches' },
                  { slug: 'docs/backends/kokoro-fastapi' },
                  { slug: 'docs/backends/llama-cpp' },
                  { slug: 'docs/backends/whisper-cpp' },
                  { slug: 'docs/backends/vllm' },
                  { slug: 'docs/backends/nemo-speech' },
                  { slug: 'docs/backends/planned' },
                ] },
                { label: 'Model profiles', items: [
                  { slug: 'docs/models', label: 'Choosing a model profile' },
                  { slug: 'docs/models/s1-mini' },
                  { slug: 'docs/models/nemotron' },
                  { slug: 'docs/models/qwen3-asr' },
                ] },
				{
					label: 'Get started',
					items: [
						{ slug: 'docs/getting-started' },
						{ slug: 'docs/guides/windows-installer' },
						{ slug: 'docs/guides/connect-a-server' },
                        { slug: 'docs/guides/saved-connections' },
                        { slug: 'docs/guides/vocabulary' },
					],
				},
				{
					label: 'Use Freehand',
					items: [
						{ slug: 'docs/guides/using-freehand' },
                        { slug: 'docs/guides/live-transcription' },
						{ slug: 'docs/guides/post-processing' },
                        { slug: 'docs/guides/languages' },
						{ slug: 'docs/guides/privacy-and-safety' },
						{ slug: 'docs/guides/troubleshooting' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ slug: 'docs/reference/protocol' },
						{ slug: 'docs/reference/shortcuts' },
                        { slug: 'docs/reference/provider-icons' },
					],
				},
				{
					label: 'Contribute',
					items: [
						{ slug: 'docs/development' },
                        { slug: 'docs/development/backend-compatibility' },
                        { slug: 'docs/development/storage' },
                        { slug: 'docs/decisions/0007-task-state-and-preference-ownership' },
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
