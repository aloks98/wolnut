<script lang="ts">
	import { onDestroy } from 'svelte';
	import { configAPI, versionAPI } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { toast } from 'svelte-sonner';
	import { createQuery } from '@tanstack/svelte-query';
	import Download from '@lucide/svelte/icons/download';
	import Upload from '@lucide/svelte/icons/upload';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import ArrowUpCircle from '@lucide/svelte/icons/arrow-up-circle';
	import ExternalLink from '@lucide/svelte/icons/external-link';

	let importing = $state(false);
	let fileInput: HTMLInputElement;
	let reloadTimer: ReturnType<typeof setTimeout> | null = null;

	const currentQuery = createQuery(() => ({
		queryKey: ['version', 'current'],
		queryFn: async () => {
			const res = await versionAPI.getCurrent();
			if (res.success && res.data) return res.data;
			throw new Error(res.error || 'Failed to fetch current version');
		},
		staleTime: Infinity
	}));

	const latestQuery = createQuery(() => ({
		queryKey: ['version', 'latest'],
		queryFn: async () => versionAPI.getLatest(),
		// Cache for 1h — GitHub's unauth API limit is 60/hr/IP.
		staleTime: 1000 * 60 * 60,
		retry: 1
	}));

	const currentVersion = $derived(currentQuery.data?.version ?? null);
	const latestVersion = $derived(latestQuery.data?.version ?? null);
	const latestUrl = $derived(latestQuery.data?.url ?? null);

	const updateAvailable = $derived(
		currentVersion !== null &&
			latestVersion !== null &&
			currentVersion !== 'dev' &&
			compareVersions(latestVersion, currentVersion) > 0
	);

	function compareVersions(a: string, b: string): number {
		const partsA = a.replace(/^v/, '').split('.').map(Number);
		const partsB = b.replace(/^v/, '').split('.').map(Number);

		for (let i = 0; i < Math.max(partsA.length, partsB.length); i++) {
			const numA = partsA[i] || 0;
			const numB = partsB[i] || 0;
			if (numA > numB) return 1;
			if (numA < numB) return -1;
		}
		return 0;
	}

	function exportConfig() {
		window.location.href = configAPI.exportUrl;
		toast.success('Configuration exported');
	}

	async function importConfig(event: Event) {
		const input = event.target as HTMLInputElement;
		const file = input.files?.[0];

		if (!file) return;

		if (!file.name.endsWith('.json')) {
			toast.error('Please select a JSON file');
			return;
		}

		importing = true;
		try {
			const res = await configAPI.import(file);
			if (res.success) {
				toast.success('Configuration imported successfully');
				reloadTimer = setTimeout(() => window.location.reload(), 1000);
			} else {
				toast.error(res.error || 'Failed to import configuration');
			}
		} finally {
			importing = false;
			input.value = '';
		}
	}

	onDestroy(() => {
		if (reloadTimer !== null) {
			clearTimeout(reloadTimer);
		}
	});
</script>

<svelte:head>
	<title>Settings · WolNUT</title>
</svelte:head>

<div class="space-y-6">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Settings</h1>
		<p class="text-muted-foreground">Backup and restore your configuration</p>
	</div>

	<!-- Update Banner -->
	{#if updateAvailable}
		<div class="rounded-lg border border-emerald-500/50 bg-emerald-500/10 p-4">
			<div class="flex items-center justify-between gap-4">
				<div class="flex items-center gap-3">
					<ArrowUpCircle class="h-5 w-5 text-emerald-500" />
					<div>
						<p class="font-medium text-emerald-500">Update available!</p>
						<p class="text-sm text-muted-foreground">
							Version <span class="font-mono tabular-nums">{latestVersion}</span> is available. You're
							on <span class="font-mono tabular-nums">{currentVersion}</span>.
						</p>
					</div>
				</div>
				{#if latestUrl}
					<Button href={latestUrl} target="_blank" rel="noopener noreferrer" size="sm" variant="outline" class="border-emerald-500/50 text-emerald-500 hover:bg-emerald-500/10">
						<ExternalLink class="h-4 w-4" />
						View Release
					</Button>
				{/if}
			</div>
		</div>
	{/if}

	<div class="grid gap-6 md:grid-cols-2">
		<!-- Export -->
		<Card.Root>
			<Card.Header>
				<Card.Title class="flex items-center gap-2">
					<Download class="h-5 w-5" />
					Export Configuration
				</Card.Title>
				<Card.Description>
					Download a backup of your devices and UPS connections
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<Button onclick={exportConfig} class="w-full">
					<Download class="h-4 w-4 mr-2" />
					Download Backup
				</Button>
			</Card.Content>
		</Card.Root>

		<!-- Import -->
		<Card.Root>
			<Card.Header>
				<Card.Title class="flex items-center gap-2">
					<Upload class="h-5 w-5" />
					Import Configuration
				</Card.Title>
				<Card.Description>
					Restore from a previously exported backup file
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<input
					type="file"
					accept=".json"
					class="hidden"
					bind:this={fileInput}
					onchange={importConfig}
				/>
				<Button
					variant="outline"
					class="w-full"
					onclick={() => fileInput.click()}
					disabled={importing}
				>
					{#if importing}
						<RefreshCw class="h-4 w-4 mr-2 animate-spin" />
						Importing...
					{:else}
						<Upload class="h-4 w-4 mr-2" />
						Select Backup File
					{/if}
				</Button>
				<p class="text-xs text-muted-foreground mt-2 text-center">
					This will replace all existing data
				</p>
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Info -->
	<Card.Root>
		<Card.Header>
			<Card.Title class="flex items-center justify-between">
				<span>About WolNUT</span>
				{#if currentVersion}
					<Badge variant="outline" class="font-mono tabular-nums">{currentVersion}</Badge>
				{/if}
			</Card.Title>
		</Card.Header>
		<Card.Content class="text-sm text-muted-foreground space-y-3">
			<p>
				WolNUT is a lightweight, self-hosted dashboard for managing Wake-on-LAN devices and monitoring UPS systems via NUT (Network UPS Tools). It provides a simple, responsive interface to wake your network devices and keep an eye on your power backup status.
			</p>
			<p>
				<a
					href="https://github.com/aloks98/wolnut"
					target="_blank"
					rel="noopener noreferrer"
					class="text-primary hover:underline"
				>
					View on GitHub
				</a>
			</p>
			<p class="text-xs pt-2 border-t border-border">
				Vibes by <a href="https://github.com/aloks98" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">@aloks98</a>, tokens by <a href="https://claude.ai" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">Claude</a>
			</p>
		</Card.Content>
	</Card.Root>
</div>
