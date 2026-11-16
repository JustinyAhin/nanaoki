const REPO = 'https://github.com/JustinyAhin/nanaoki';

const SITE = {
	// Use the public origin. Never derive it from a preview request.
	origin: 'https://nanaoki.com',
	name: 'nanaoki',
	robots: 'index, follow',
	repo: REPO,
	releases: `${REPO}/releases`,
	license: `${REPO}/blob/main/LICENSE`,
	author: { name: 'Justin Ahinon', url: 'https://segbedji.com' },
	ogImage: {
		// Bump the version when og.png changes, so that social sites fetch the new card.
		path: '/assets/og.png?v=2',
		width: 1200,
		height: 630,
		alt: 'nanaoki. Your agent already fixed this. Seven pale lines fall away; an eighth, in gold, rises.'
	}
};

const INSTALL = {
	brew: ['brew tap JustinyAhin/tap', 'brew trust JustinyAhin/tap', 'brew install nanaoki'],
	curl: ['curl -fsSL https://raw.githubusercontent.com/JustinyAhin/nanaoki/main/install.sh | sh'],
	go: ['go install github.com/JustinyAhin/nanaoki/cmd/nanaoki@latest']
};

export { SITE, INSTALL };
