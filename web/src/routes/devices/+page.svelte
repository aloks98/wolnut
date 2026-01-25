<script lang="ts">
	import { onMount } from 'svelte';
	import { deviceAPI } from '$lib/api';
	import type { Device } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { toast } from 'svelte-sonner';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Power from '@lucide/svelte/icons/power';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';

	let devices = $state<Device[]>([]);
	let loading = $state(true);
	let dialogOpen = $state(false);
	let submitting = $state(false);
	let deletingId = $state<string | null>(null);
	let wakingId = $state<string | null>(null);

	// Form state
	let name = $state('');
	let mac = $state('');

	async function loadDevices() {
		const res = await deviceAPI.getAll();
		if (res.success && res.data) {
			devices = res.data;
		}
		loading = false;
	}

	async function addDevice() {
		if (!name.trim() || !mac.trim()) {
			toast.error('Name and MAC address are required');
			return;
		}

		submitting = true;
		const res = await deviceAPI.create({ name: name.trim(), mac: mac.trim() });

		if (res.success && res.data) {
			devices = [...devices, res.data];
			toast.success(`Device "${name}" added`);
			dialogOpen = false;
			name = '';
			mac = '';
		} else {
			toast.error(res.error || 'Failed to add device');
		}
		submitting = false;
	}

	async function deleteDevice(device: Device) {
		deletingId = device.id;
		const res = await deviceAPI.delete(device.id);

		if (res.success) {
			devices = devices.filter(d => d.id !== device.id);
			toast.success(`Device "${device.name}" deleted`);
		} else {
			toast.error(res.error || 'Failed to delete device');
		}
		deletingId = null;
	}

	async function wakeDevice(device: Device) {
		wakingId = device.id;
		const res = await deviceAPI.wake(device.id);

		if (res.success) {
			toast.success(`Wake packet sent to ${device.name}`);
		} else {
			toast.error(res.error || 'Failed to wake device');
		}
		wakingId = null;
	}

	onMount(() => {
		loadDevices();
	});
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">Devices</h1>
			<p class="text-muted-foreground">Manage your Wake-on-LAN devices</p>
		</div>
		<Dialog.Root bind:open={dialogOpen}>
			<Dialog.Trigger>
				{#snippet child({ props })}
					<Button {...props}>
						<Plus class="h-4 w-4 mr-1" />
						Add Device
					</Button>
				{/snippet}
			</Dialog.Trigger>
			<Dialog.Content class="sm:max-w-md">
				<Dialog.Header>
					<Dialog.Title>Add Device</Dialog.Title>
					<Dialog.Description>
						Add a new device for Wake-on-LAN
					</Dialog.Description>
				</Dialog.Header>
				<form onsubmit={(e) => { e.preventDefault(); addDevice(); }} class="space-y-4">
					<div class="space-y-2">
						<label for="name" class="text-sm font-medium">Name</label>
						<Input
							id="name"
							placeholder="e.g., Desktop PC"
							bind:value={name}
							required
						/>
					</div>
					<div class="space-y-2">
						<label for="mac" class="text-sm font-medium">MAC Address</label>
						<Input
							id="mac"
							placeholder="AA:BB:CC:DD:EE:FF"
							bind:value={mac}
							class="font-mono"
							required
						/>
						<p class="text-xs text-muted-foreground">
							Format: AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF
						</p>
					</div>
					<Dialog.Footer>
						<Button type="button" variant="outline" onclick={() => dialogOpen = false}>
							Cancel
						</Button>
						<Button type="submit" disabled={submitting}>
							{#if submitting}
								<RefreshCw class="h-4 w-4 mr-1 animate-spin" />
							{/if}
							Add Device
						</Button>
					</Dialog.Footer>
				</form>
			</Dialog.Content>
		</Dialog.Root>
	</div>

	{#if loading}
		<div class="space-y-3">
			{#each [1, 2, 3] as _}
				<Card.Root>
					<Card.Content class="p-4">
						<div class="animate-pulse flex items-center justify-between">
							<div class="space-y-2">
								<div class="h-4 bg-muted rounded w-32"></div>
								<div class="h-3 bg-muted rounded w-40"></div>
							</div>
							<div class="h-8 bg-muted rounded w-20"></div>
						</div>
					</Card.Content>
				</Card.Root>
			{/each}
		</div>
	{:else if devices.length === 0}
		<Card.Root>
			<Card.Content class="p-12 text-center text-muted-foreground">
				<Power class="h-12 w-12 mx-auto mb-4 opacity-50" />
				<p class="mb-2">No devices configured yet.</p>
				<p class="text-sm">Click "Add Device" to get started.</p>
			</Card.Content>
		</Card.Root>
	{:else}
		<div class="space-y-3">
			{#each devices as device}
				<Card.Root>
					<Card.Content class="p-4 flex items-center justify-between">
						<div>
							<h3 class="font-medium">{device.name}</h3>
							<p class="text-sm text-muted-foreground font-mono">{device.mac}</p>
						</div>
						<div class="flex items-center gap-2">
							<Button
								size="sm"
								onclick={() => wakeDevice(device)}
								disabled={wakingId === device.id}
							>
								{#if wakingId === device.id}
									<RefreshCw class="h-4 w-4 mr-1 animate-spin" />
								{:else}
									<Power class="h-4 w-4 mr-1" />
								{/if}
								Wake
							</Button>
							<Button
								size="sm"
								variant="destructive"
								onclick={() => deleteDevice(device)}
								disabled={deletingId === device.id}
							>
								{#if deletingId === device.id}
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
