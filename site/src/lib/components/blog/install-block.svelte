<script lang="ts">
	import InstallSwitcher from '../install-switcher.svelte';
</script>

<!-- The same switcher as the home page, so the commands never drift. -->
<div class="install">
	<InstallSwitcher />
</div>

<style>
	.install {
		margin: 2em 0;
		font: 400 16px/1.5 var(--sans);
	}
</style>
