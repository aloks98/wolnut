<script lang="ts">
	import { configAPI } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { toast } from 'svelte-sonner';
	import Download from '@lucide/svelte/icons/download';
	import Upload from '@lucide/svelte/icons/upload';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';

	let importing = $state(false);
	let fileInput: HTMLInputElement;

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
		const res = await configAPI.import(file);

		if (res.success) {
			toast.success('Configuration imported successfully');
			// Reload the page to reflect changes
			setTimeout(() => window.location.reload(), 1000);
		} else {
			toast.error(res.error || 'Failed to import configuration');
		}

		importing = false;
		input.value = '';
	}
</script>

<div class="space-y-6">
	<div>
		<h1 class="text-2xl font-bold">Settings</h1>
		<p class="text-muted-foreground">Backup and restore your configuration</p>
	</div>

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
			<Card.Title>About WoL-NUT</Card.Title>
		</Card.Header>
		<Card.Content class="text-sm text-muted-foreground space-y-3">
			<p>
				WoL-NUT is a lightweight, self-hosted dashboard for managing Wake-on-LAN devices and monitoring UPS systems via NUT (Network UPS Tools). It provides a simple, responsive interface to wake your network devices and keep an eye on your power backup status.
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
