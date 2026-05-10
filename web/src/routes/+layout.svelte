<script module lang="ts">
	import { browser } from '$app/environment';
	import { QueryClient } from '@tanstack/svelte-query';

	// Module-scoped so the cache survives HMR and the instance is stable
	// across layout remounts.
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				enabled: browser,
				staleTime: 1000 * 30,
				refetchOnWindowFocus: true
			}
		}
	});
</script>

<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from '$lib/components/ui/sonner';
	import { Nav } from '$lib/components/layout';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import * as Tooltip from '$lib/components/ui/tooltip';

	let { children } = $props();
</script>

<svelte:head>
	<title>WolNUT</title>
</svelte:head>

<ModeWatcher defaultMode="dark" />
<Toaster richColors position="top-right" />

<Tooltip.Provider>
	<QueryClientProvider client={queryClient}>
		<div class="min-h-screen bg-background">
			<Nav />
			<main class="mx-auto max-w-6xl px-4 py-6">
				{@render children()}
			</main>
		</div>
	</QueryClientProvider>
</Tooltip.Provider>
