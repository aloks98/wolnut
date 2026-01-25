<script lang="ts">
	import { onMount } from 'svelte';
	import { upsAPI } from '$lib/api';
	import type { UPSEntry } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { toast } from 'svelte-sonner';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Zap from '@lucide/svelte/icons/zap';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';

	let upsList = $state<UPSEntry[]>([]);
	let loading = $state(true);
	let addDialogOpen = $state(false);
	let editDialogOpen = $state(false);
	let submitting = $state(false);
	let deletingId = $state<string | null>(null);

	// Form state
	let name = $state('');
	let host = $state('');
	let upsName = $state('ups');
	let editingId = $state<string | null>(null);

	async function loadUPS() {
		const res = await upsAPI.getAll();
		if (res.success && res.data) {
			upsList = res.data;
		}
		loading = false;
	}

	function openAddDialog() {
		name = '';
		host = '';
		upsName = 'ups';
		editingId = null;
		addDialogOpen = true;
	}

	function openEditDialog(ups: UPSEntry) {
		name = ups.name;
		host = ups.host;
		upsName = ups.ups_name;
		editingId = ups.id;
		editDialogOpen = true;
	}

	async function saveUPS() {
		if (!name.trim() || !host.trim() || !upsName.trim()) {
			toast.error('All fields are required');
			return;
		}

		submitting = true;

		const upsData = {
			name: name.trim(),
			host: host.trim(),
			ups_name: upsName.trim()
		};

		if (editingId) {
			// Update existing
			const res = await upsAPI.update(editingId, upsData);
			if (res.success) {
				await loadUPS();
				toast.success(`UPS "${name}" updated`);
				editDialogOpen = false;
			} else {
				toast.error(res.error || 'Failed to update UPS');
			}
		} else {
			// Create new
			const res = await upsAPI.create(upsData);
			if (res.success && res.data) {
				upsList = [...upsList, res.data];
				toast.success(`UPS "${name}" added`);
				addDialogOpen = false;
			} else {
				toast.error(res.error || 'Failed to add UPS');
			}
		}

		submitting = false;
	}

	async function deleteUPS(ups: UPSEntry) {
		deletingId = ups.id;
		const res = await upsAPI.delete(ups.id);

		if (res.success) {
			upsList = upsList.filter(u => u.id !== ups.id);
			toast.success(`UPS "${ups.name}" deleted`);
		} else {
			toast.error(res.error || 'Failed to delete UPS');
		}
		deletingId = null;
	}

	onMount(() => {
		loadUPS();
	});
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">UPS Connections</h1>
			<p class="text-muted-foreground">Manage your NUT UPS server connections</p>
		</div>
		<Button onclick={openAddDialog}>
			<Plus class="h-4 w-4 mr-1" />
			Add UPS
		</Button>
	</div>

	<!-- Add Dialog -->
	<Dialog.Root bind:open={addDialogOpen}>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header>
				<Dialog.Title>Add UPS Connection</Dialog.Title>
				<Dialog.Description>
					Connect to a NUT (Network UPS Tools) server
				</Dialog.Description>
			</Dialog.Header>
			<form onsubmit={(e) => { e.preventDefault(); saveUPS(); }} class="space-y-4">
				<div class="space-y-2">
					<label for="name" class="text-sm font-medium">Display Name</label>
					<Input
						id="name"
						placeholder="e.g., Office UPS"
						bind:value={name}
						required
					/>
				</div>
				<div class="space-y-2">
					<label for="host" class="text-sm font-medium">NUT Server Host</label>
					<Input
						id="host"
						placeholder="localhost:3493"
						bind:value={host}
						class="font-mono"
						required
					/>
					<p class="text-xs text-muted-foreground">
						Host and port of the NUT server (default port: 3493)
					</p>
				</div>
				<div class="space-y-2">
					<label for="upsName" class="text-sm font-medium">UPS Name</label>
					<Input
						id="upsName"
						placeholder="ups"
						bind:value={upsName}
						class="font-mono"
						required
					/>
					<p class="text-xs text-muted-foreground">
						UPS identifier as configured in NUT (usually "ups")
					</p>
				</div>
				<Dialog.Footer>
					<Button type="button" variant="outline" onclick={() => addDialogOpen = false}>
						Cancel
					</Button>
					<Button type="submit" disabled={submitting}>
						{#if submitting}
							<RefreshCw class="h-4 w-4 mr-1 animate-spin" />
						{/if}
						Add UPS
					</Button>
				</Dialog.Footer>
			</form>
		</Dialog.Content>
	</Dialog.Root>

	<!-- Edit Dialog -->
	<Dialog.Root bind:open={editDialogOpen}>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header>
				<Dialog.Title>Edit UPS Connection</Dialog.Title>
				<Dialog.Description>
					Update UPS connection information
				</Dialog.Description>
			</Dialog.Header>
			<form onsubmit={(e) => { e.preventDefault(); saveUPS(); }} class="space-y-4">
				<div class="space-y-2">
					<label for="edit-name" class="text-sm font-medium">Display Name</label>
					<Input
						id="edit-name"
						placeholder="e.g., Office UPS"
						bind:value={name}
						required
					/>
				</div>
				<div class="space-y-2">
					<label for="edit-host" class="text-sm font-medium">NUT Server Host</label>
					<Input
						id="edit-host"
						placeholder="localhost:3493"
						bind:value={host}
						class="font-mono"
						required
					/>
					<p class="text-xs text-muted-foreground">
						Host and port of the NUT server (default port: 3493)
					</p>
				</div>
				<div class="space-y-2">
					<label for="edit-upsName" class="text-sm font-medium">UPS Name</label>
					<Input
						id="edit-upsName"
						placeholder="ups"
						bind:value={upsName}
						class="font-mono"
						required
					/>
					<p class="text-xs text-muted-foreground">
						UPS identifier as configured in NUT (usually "ups")
					</p>
				</div>
				<Dialog.Footer>
					<Button type="button" variant="outline" onclick={() => editDialogOpen = false}>
						Cancel
					</Button>
					<Button type="submit" disabled={submitting}>
						{#if submitting}
							<RefreshCw class="h-4 w-4 mr-1 animate-spin" />
						{/if}
						Save Changes
					</Button>
				</Dialog.Footer>
			</form>
		</Dialog.Content>
	</Dialog.Root>

	{#if loading}
		<div class="space-y-2">
			{#each [1, 2] as _}
				<Card.Root>
					<Card.Content class="py-3 px-4">
						<div class="animate-pulse flex items-center justify-between">
							<div class="space-y-2">
								<div class="h-4 bg-muted rounded w-32"></div>
								<div class="h-3 bg-muted rounded w-48"></div>
							</div>
							<div class="h-8 bg-muted rounded w-10"></div>
						</div>
					</Card.Content>
				</Card.Root>
			{/each}
		</div>
	{:else if upsList.length === 0}
		<Card.Root>
			<Card.Content class="p-12 text-center text-muted-foreground">
				<Zap class="h-12 w-12 mx-auto mb-4 opacity-50" />
				<p class="mb-2">No UPS connections configured yet.</p>
				<p class="text-sm">Click "Add UPS" to connect to a NUT server.</p>
			</Card.Content>
		</Card.Root>
	{:else}
		<div class="space-y-2">
			{#each upsList as ups}
				<Card.Root>
					<Card.Content class="py-3 px-4 flex items-center justify-between">
						<div>
							<h3 class="font-medium">{ups.name}</h3>
							<p class="text-sm text-muted-foreground font-mono">
								{ups.host} / {ups.ups_name}
							</p>
						</div>
						<div class="flex items-center gap-2">
							<Button
								size="sm"
								variant="outline"
								onclick={() => openEditDialog(ups)}
							>
								<Pencil class="h-4 w-4" />
							</Button>
							<Button
								size="sm"
								variant="destructive"
								onclick={() => deleteUPS(ups)}
								disabled={deletingId === ups.id}
							>
								{#if deletingId === ups.id}
									<RefreshCw class="h-4 w-4 animate-spin" />
								{:else}
									<Trash2 class="h-4 w-4" />
								{/if}
							</Button>
						</div>
					</Card.Content>
				</Card.Root>
			{/each}
		</div>
	{/if}
</div>
