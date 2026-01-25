<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from '$lib/components/ui/sonner';
	import { Nav } from '$lib/components/layout';

	let { children } = $props();
</script>

<svelte:head>
	<title>WoL-NUT</title>
</svelte:head>

<ModeWatcher defaultMode="dark" />
<Toaster richColors position="top-right" />

<div class="min-h-screen bg-background">
	<Nav />
	<main class="mx-auto max-w-6xl px-4 py-6">
		{@render children()}
	</main>
</div>
