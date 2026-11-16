import { INSTALL } from './site';

type InstallMethod = keyof typeof INSTALL;

// The home page and /cases share this key, so the visitor's choice follows them.
const STORAGE_KEY = 'nanaoki-install';
const METHODS = Object.keys(INSTALL) as InstallMethod[];

const install = $state<{ method: InstallMethod }>({ method: 'brew' });

const isMethod = (value: string | null): value is InstallMethod =>
	METHODS.includes(value as InstallMethod);

const loadInstallMethod = () => {
	try {
		const saved = localStorage.getItem(STORAGE_KEY);
		if (isMethod(saved)) install.method = saved;
	} catch {
		// Storage can be blocked. The default tab still works.
	}
};

const selectInstallMethod = (method: InstallMethod) => {
	install.method = method;

	try {
		localStorage.setItem(STORAGE_KEY, method);
	} catch {
		// Storage can be blocked. The choice lasts for this page view.
	}
};

export { METHODS, install, loadInstallMethod, selectInstallMethod };
export type { InstallMethod };
