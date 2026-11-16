// One live region for the whole page announces copy results to screen readers.
const copyStatus = $state({ text: '' });

const legacyCopy = (text: string) => {
	const area = document.createElement('textarea');

	area.value = text;
	area.style.position = 'fixed';
	area.style.opacity = '0';
	document.body.appendChild(area);
	area.select();

	try {
		return document.execCommand('copy');
	} catch {
		return false;
	} finally {
		area.remove();
	}
};

const copyText = async (text: string) => {
	if (navigator.clipboard && window.isSecureContext) {
		try {
			await navigator.clipboard.writeText(text);
			return true;
		} catch {
			return legacyCopy(text);
		}
	}

	return legacyCopy(text);
};

export { copyStatus, copyText };
