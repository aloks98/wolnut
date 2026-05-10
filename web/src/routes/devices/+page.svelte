<script lang="ts">
	import { deviceAPI } from '$lib/api';
	import type { DeviceStatus } from '$lib/api';
	import { deviceStatusBadgeClass, deviceStatusLabel } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field';
	import * as Table from '$lib/components/ui/table';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { Badge } from '$lib/components/ui/badge';
	import { Input } from '$lib/components/ui/input';
	import { toast } from 'svelte-sonner';
	import { superForm, defaults } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { deviceSchema } from '$lib/schemas';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Power from '@lucide/svelte/icons/power';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';

	let addDialogOpen = $state(false);
	let editDialogOpen = $state(false);
	let deletingId = $state<string | null>(null);
	let wakingId = $state<string | null>(null);
	let editingId = $state<string | null>(null);
	let pendingDelete = $state<DeviceStatus | null>(null);

	const queryClient = useQueryClient();

	// Device status query with auto-refresh
	const devicesQuery = createQuery(() => ({
		queryKey: ['devices', 'status'],
		queryFn: async () => {
			const res = await deviceAPI.getStatus();
			if (res.success && res.data) {
				return res.data;
			}
			throw new Error(res.error || 'Failed to fetch devices');
		},
		refetchInterval: 30000,
		refetchIntervalInBackground: false
	}));

	// Derived state
	const devices = $derived(devicesQuery.data ?? []);
	const loading = $derived(devicesQuery.isLoading);

	async function invalidateDevices() {
		await queryClient.invalidateQueries({ queryKey: ['devices', 'status'] });
	}

	// Superform for Add dialog
	const addForm = superForm(defaults({ name: '', mac: '', ip: '' }, zod4(deviceSchema)), {
		SPA: true,
		validators: zod4(deviceSchema),
		onUpdate: async ({ form }) => {
			if (!form.valid) return;

			const res = await deviceAPI.create({
				name: form.data.name,
				mac: form.data.mac,
				ip: form.data.ip || undefined
			});

			if (res.success) {
				await invalidateDevices();
				toast.success(`Device "${form.data.name}" added`);
				addDialogOpen = false;
				addForm.reset();
			} else {
				toast.error(res.error || 'Failed to add device');
			}
		}
	});

	// Superform for Edit dialog
	const editForm = superForm(defaults({ name: '', mac: '', ip: '' }, zod4(deviceSchema)), {
		SPA: true,
		validators: zod4(deviceSchema),
		onUpdate: async ({ form }) => {
			if (!form.valid || !editingId) return;

			const res = await deviceAPI.update(editingId, {
				name: form.data.name,
				mac: form.data.mac,
				ip: form.data.ip || undefined
			});

			if (res.success) {
				await invalidateDevices();
				toast.success(`Device "${form.data.name}" updated`);
				editDialogOpen = false;
				editForm.reset();
				editingId = null;
			} else {
				toast.error(res.error || 'Failed to update device');
			}
		}
	});

	const {
		form: addFormData,
		errors: addErrors,
		enhance: addEnhance,
		submitting: addSubmitting
	} = addForm;
	const {
		form: editFormData,
		errors: editErrors,
		enhance: editEnhance,
		submitting: editSubmitting
	} = editForm;

	function openAddDialog() {
		addForm.reset();
		addDialogOpen = true;
	}

	function openEditDialog(device: DeviceStatus) {
		editingId = device.id;
		editForm.reset({
			data: {
				name: device.name,
				mac: device.mac,
				ip: device.ip || ''
			}
		});
		editDialogOpen = true;
	}

	function requestDelete(device: DeviceStatus) {
		pendingDelete = device;
	}

	async function confirmDelete() {
		const device = pendingDelete;
		if (!device) return;
		deletingId = device.id;
		pendingDelete = null;
		try {
			const res = await deviceAPI.delete(device.id);
			if (res.success) {
				await invalidateDevices();
				toast.success(`Device "${device.name}" deleted`);
			} else {
				toast.error(res.error || 'Failed to delete device');
			}
		} finally {
			deletingId = null;
		}
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

</script>

<svelte:head>
	<title>Devices · WolNUT</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Devices</h1>
			<p class="text-muted-foreground">Manage your Wake-on-LAN devices</p>
		</div>
		<Button onclick={openAddDialog}>
			<Plus class="h-4 w-4" />
			Add Device
		</Button>
	</div>

	<!-- Add Dialog -->
	<Dialog.Root bind:open={addDialogOpen}>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header>
				<Dialog.Title>Add Device</Dialog.Title>
				<Dialog.Description>Add a new device for Wake-on-LAN</Dialog.Description>
			</Dialog.Header>
			<form method="POST" use:addEnhance class="space-y-4">
				<Field.Field data-invalid={$addErrors.name ? true : undefined}>
					<Field.Label for="add-name">Name</Field.Label>
					<Input
						id="add-name"
						placeholder="e.g., Desktop PC"
						bind:value={$addFormData.name}
						aria-invalid={$addErrors.name ? 'true' : undefined}
						aria-describedby={$addErrors.name ? 'add-name-error' : undefined}
					/>
					{#if $addErrors.name}
						<Field.Error id="add-name-error">{$addErrors.name}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field data-invalid={$addErrors.mac ? true : undefined}>
					<Field.Label for="add-mac">MAC Address</Field.Label>
					<Input
						id="add-mac"
						placeholder="AA:BB:CC:DD:EE:FF"
						bind:value={$addFormData.mac}
						class="font-mono"
						aria-invalid={$addErrors.mac ? 'true' : undefined}
						aria-describedby={$addErrors.mac ? 'add-mac-error' : 'add-mac-help'}
					/>
					<Field.Description id="add-mac-help">Format: AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF</Field.Description>
					{#if $addErrors.mac}
						<Field.Error id="add-mac-error">{$addErrors.mac}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field data-invalid={$addErrors.ip ? true : undefined}>
					<Field.Label for="add-ip">IP Address (optional)</Field.Label>
					<Input
						id="add-ip"
						placeholder="192.168.1.100"
						bind:value={$addFormData.ip}
						class="font-mono"
						aria-invalid={$addErrors.ip ? 'true' : undefined}
						aria-describedby={$addErrors.ip ? 'add-ip-error' : 'add-ip-help'}
					/>
					<Field.Description id="add-ip-help">Used to check online status</Field.Description>
					{#if $addErrors.ip}
						<Field.Error id="add-ip-error">{$addErrors.ip}</Field.Error>
					{/if}
				</Field.Field>
				<Dialog.Footer>
					<Button type="button" variant="outline" onclick={() => (addDialogOpen = false)}>
						Cancel
					</Button>
					<Button type="submit" disabled={$addSubmitting}>
						{#if $addSubmitting}
							<RefreshCw class="h-4 w-4 animate-spin" />
						{/if}
						Add Device
					</Button>
				</Dialog.Footer>
			</form>
		</Dialog.Content>
	</Dialog.Root>

	<!-- Delete Confirm Dialog -->
	<Dialog.Root
		open={pendingDelete !== null}
		onOpenChange={(o) => {
			if (!o) pendingDelete = null;
		}}
	>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header>
				<Dialog.Title>Delete device?</Dialog.Title>
				<Dialog.Description>
					"{pendingDelete?.name}" will be removed permanently. This cannot be undone.
				</Dialog.Description>
			</Dialog.Header>
			<Dialog.Footer>
				<Button type="button" variant="outline" onclick={() => (pendingDelete = null)}>
					Cancel
				</Button>
				<Button type="button" variant="destructive" onclick={confirmDelete}>
					Delete
				</Button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<!-- Edit Dialog -->
	<Dialog.Root bind:open={editDialogOpen}>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header>
				<Dialog.Title>Edit Device</Dialog.Title>
				<Dialog.Description>Update device information</Dialog.Description>
			</Dialog.Header>
			<form method="POST" use:editEnhance class="space-y-4">
				<Field.Field data-invalid={$editErrors.name ? true : undefined}>
					<Field.Label for="edit-name">Name</Field.Label>
					<Input
						id="edit-name"
						placeholder="e.g., Desktop PC"
						bind:value={$editFormData.name}
						aria-invalid={$editErrors.name ? 'true' : undefined}
						aria-describedby={$editErrors.name ? 'edit-name-error' : undefined}
					/>
					{#if $editErrors.name}
						<Field.Error id="edit-name-error">{$editErrors.name}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field data-invalid={$editErrors.mac ? true : undefined}>
					<Field.Label for="edit-mac">MAC Address</Field.Label>
					<Input
						id="edit-mac"
						placeholder="AA:BB:CC:DD:EE:FF"
						bind:value={$editFormData.mac}
						class="font-mono"
						aria-invalid={$editErrors.mac ? 'true' : undefined}
						aria-describedby={$editErrors.mac ? 'edit-mac-error' : 'edit-mac-help'}
					/>
					<Field.Description id="edit-mac-help">Format: AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF</Field.Description>
					{#if $editErrors.mac}
						<Field.Error id="edit-mac-error">{$editErrors.mac}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field data-invalid={$editErrors.ip ? true : undefined}>
					<Field.Label for="edit-ip">IP Address (optional)</Field.Label>
					<Input
						id="edit-ip"
						placeholder="192.168.1.100"
						bind:value={$editFormData.ip}
						class="font-mono"
						aria-invalid={$editErrors.ip ? 'true' : undefined}
						aria-describedby={$editErrors.ip ? 'edit-ip-error' : 'edit-ip-help'}
					/>
					<Field.Description id="edit-ip-help">Used to check online status</Field.Description>
					{#if $editErrors.ip}
						<Field.Error id="edit-ip-error">{$editErrors.ip}</Field.Error>
					{/if}
				</Field.Field>
				<Dialog.Footer>
					<Button type="button" variant="outline" onclick={() => (editDialogOpen = false)}>
						Cancel
					</Button>
					<Button type="submit" disabled={$editSubmitting}>
						{#if $editSubmitting}
							<RefreshCw class="h-4 w-4 animate-spin" />
						{/if}
						Save Changes
					</Button>
				</Dialog.Footer>
			</form>
		</Dialog.Content>
	</Dialog.Root>

	{#if loading}
		<div class="rounded-md border">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="w-24">Status</Table.Head>
						<Table.Head>Name</Table.Head>
						<Table.Head>MAC Address</Table.Head>
						<Table.Head>IP Address</Table.Head>
						<Table.Head class="w-32 text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each [1, 2, 3] as _}
						<Table.Row>
							<Table.Cell><div class="h-5 w-16 bg-muted rounded animate-pulse"></div></Table.Cell>
							<Table.Cell><div class="h-4 w-24 bg-muted rounded animate-pulse"></div></Table.Cell>
							<Table.Cell><div class="h-4 w-32 bg-muted rounded animate-pulse"></div></Table.Cell>
							<Table.Cell><div class="h-4 w-24 bg-muted rounded animate-pulse"></div></Table.Cell>
							<Table.Cell><div class="h-8 w-24 bg-muted rounded animate-pulse ml-auto"></div></Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
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
		<div class="rounded-md border">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="w-24">Status</Table.Head>
						<Table.Head>Name</Table.Head>
						<Table.Head>MAC Address</Table.Head>
						<Table.Head>IP Address</Table.Head>
						<Table.Head class="w-32 text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each devices as device (device.id)}
						<Table.Row>
							<Table.Cell>
								<Badge variant="outline" class={deviceStatusBadgeClass(device.online)}>
									{deviceStatusLabel(device.online)}
								</Badge>
							</Table.Cell>
							<Table.Cell class="font-medium">{device.name}</Table.Cell>
							<Table.Cell class="font-mono text-muted-foreground">{device.mac}</Table.Cell>
							<Table.Cell class="font-mono text-muted-foreground">{device.ip || '—'}</Table.Cell>
							<Table.Cell class="text-right">
								<div class="flex justify-end gap-1">
									<Tooltip.Root>
										<Tooltip.Trigger>
											<Button
												size="icon"
												variant="outline"
												onclick={() => wakeDevice(device)}
												disabled={wakingId === device.id}
												aria-label="Wake {device.name}"
											>
												{#if wakingId === device.id}
													<RefreshCw class="h-4 w-4 animate-spin" />
												{:else}
													<Power class="h-4 w-4" />
												{/if}
											</Button>
										</Tooltip.Trigger>
										<Tooltip.Content>
											<p>Wake {device.name}</p>
										</Tooltip.Content>
									</Tooltip.Root>
									<Tooltip.Root>
										<Tooltip.Trigger>
											<Button
												size="icon"
												variant="outline"
												onclick={() => openEditDialog(device)}
												aria-label="Edit {device.name}"
											>
												<Pencil class="h-4 w-4" />
											</Button>
										</Tooltip.Trigger>
										<Tooltip.Content>
											<p>Edit device</p>
										</Tooltip.Content>
									</Tooltip.Root>
									<Tooltip.Root>
										<Tooltip.Trigger>
											<Button
												size="icon"
												variant="destructive"
												onclick={() => requestDelete(device)}
												disabled={deletingId === device.id}
												aria-label="Delete {device.name}"
											>
												{#if deletingId === device.id}
													<RefreshCw class="h-4 w-4 animate-spin" />
												{:else}
													<Trash2 class="h-4 w-4" />
												{/if}
											</Button>
										</Tooltip.Trigger>
										<Tooltip.Content>
											<p>Delete device</p>
										</Tooltip.Content>
									</Tooltip.Root>
								</div>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</div>
	{/if}
</div>
