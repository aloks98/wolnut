<script lang="ts">
	import { onMount } from 'svelte';
	import { deviceAPI } from '$lib/api';
	import type { DeviceStatus } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { toast } from 'svelte-sonner';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Power from '@lucide/svelte/icons/power';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Circle from '@lucide/svelte/icons/circle';

	let devices = $state<DeviceStatus[]>([]);
	let loading = $state(true);
	let addDialogOpen = $state(false);
	let editDialogOpen = $state(false);
	let submitting = $state(false);
	let deletingId = $state<string | null>(null);
	let wakingId = $state<string | null>(null);

	// Form state
	let name = $state('');
	let mac = $state('');
	let ip = $state('');
	let editingId = $state<string | null>(null);

	async function loadDevices() {
		const res = await deviceAPI.getStatus();
		if (res.success && res.data) {
			devices = res.data;
		}
		loading = false;
	}

	function openAddDialog() {
		name = '';
		mac = '';
		ip = '';
		editingId = null;
		addDialogOpen = true;
	}

	function openEditDialog(device: DeviceStatus) {
		name = device.name;
		mac = device.mac;
		ip = device.ip || '';
		editingId = device.id;
		editDialogOpen = true;
	}

	async function saveDevice() {
		if (!name.trim() || !mac.trim()) {
			toast.error('Name and MAC address are required');
			return;
		}

		submitting = true;

		const deviceData = {
			name: name.trim(),
			mac: mac.trim(),
			ip: ip.trim() || undefined
		};

		if (editingId) {
			// Update existing
			const res = await deviceAPI.update(editingId, deviceData);
			if (res.success) {
				await loadDevices();
				toast.success(`Device "${name}" updated`);
				editDialogOpen = false;
			} else {
				toast.error(res.error || 'Failed to update device');
			}
		} else {
			// Create new
			const res = await deviceAPI.create(deviceData);
			if (res.success) {
				await loadDevices();
				toast.success(`Device "${name}" added`);
				addDialogOpen = false;
			} else {
				toast.error(res.error || 'Failed to add device');
			}
		}

		submitting = false;
	}

	async function deleteDevice(device: DeviceStatus) {
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

	async function wakeDevice(device: DeviceStatus) {
		wakingId = device.id;
		const res = await deviceAPI.wake(device.id);

		if (res.success) {
			toast.success(`Wake packet sent to ${device.name}`);
		} else {
			toast.error(res.error || 'Failed to wake device');
		}
		wakingId = null;
	}

	function getStatusColor(online: boolean | null): string {
		if (online === null) return 'text-muted-foreground';
		return online ? 'text-emerald-500' : 'text-red-500';
	}

	function getStatusTitle(online: boolean | null): string {
		if (online === null) return 'No IP configured';
		return online ? 'Online' : 'Offline';
	}

	onMount(() => {
		loadDevices();

		const interval = setInterval(() => {
			if (!document.hidden) {
				loadDevices();
			}
		}, 30000);

		return () => clearInterval(interval);
	});
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">Devices</h1>
			<p class="text-muted-foreground">Manage your Wake-on-LAN devices</p>
		</div>
		<Button onclick={openAddDialog}>
			<Plus class="h-4 w-4 mr-1" />
			Add Device
		</Button>
	</div>

	<!-- Add Dialog -->
	<Dialog.Root bind:open={addDialogOpen}>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header>
				<Dialog.Title>Add Device</Dialog.Title>
				<Dialog.Description>
					Add a new device for Wake-on-LAN
				</Dialog.Description>
			</Dialog.Header>
			<form onsubmit={(e) => { e.preventDefault(); saveDevice(); }} class="space-y-4">
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
				<div class="space-y-2">
					<label for="ip" class="text-sm font-medium">IP Address (optional)</label>
					<Input
						id="ip"
						placeholder="192.168.1.100"
						bind:value={ip}
						class="font-mono"
					/>
					<p class="text-xs text-muted-foreground">
						Used to check online status
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
						Add Device
					</Button>
				</Dialog.Footer>
			</form>
		</Dialog.Content>
	</Dialog.Root>

	<!-- Edit Dialog -->
	<Dialog.Root bind:open={editDialogOpen}>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header>
				<Dialog.Title>Edit Device</Dialog.Title>
				<Dialog.Description>
					Update device information
				</Dialog.Description>
			</Dialog.Header>
			<form onsubmit={(e) => { e.preventDefault(); saveDevice(); }} class="space-y-4">
				<div class="space-y-2">
					<label for="edit-name" class="text-sm font-medium">Name</label>
					<Input
						id="edit-name"
						placeholder="e.g., Desktop PC"
						bind:value={name}
						required
					/>
				</div>
				<div class="space-y-2">
					<label for="edit-mac" class="text-sm font-medium">MAC Address</label>
					<Input
						id="edit-mac"
						placeholder="AA:BB:CC:DD:EE:FF"
						bind:value={mac}
						class="font-mono"
						required
					/>
					<p class="text-xs text-muted-foreground">
						Format: AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF
					</p>
				</div>
				<div class="space-y-2">
					<label for="edit-ip" class="text-sm font-medium">IP Address (optional)</label>
					<Input
						id="edit-ip"
						placeholder="192.168.1.100"
						bind:value={ip}
						class="font-mono"
					/>
					<p class="text-xs text-muted-foreground">
						Used to check online status
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
			{#each [1, 2, 3] as _}
				<Card.Root>
					<Card.Content class="py-3 px-4">
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
		<div class="space-y-2">
			{#each devices as device}
				<Card.Root>
					<Card.Content class="py-3 px-4 flex items-center justify-between">
						<div class="flex items-center gap-3">
							<Circle
								class="h-3 w-3 fill-current {getStatusColor(device.online)}"
								title={getStatusTitle(device.online)}
							/>
							<div>
								<h3 class="font-medium">{device.name}</h3>
								<p class="text-sm text-muted-foreground font-mono">
									{device.mac}
									{#if device.ip}
										<span class="text-muted-foreground/60"> · {device.ip}</span>
									{/if}
								</p>
							</div>
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
								variant="outline"
								onclick={() => openEditDialog(device)}
							>
								<Pencil class="h-4 w-4" />
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
