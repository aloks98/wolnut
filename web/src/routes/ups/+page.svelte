<script lang="ts">
	import { onMount } from 'svelte';
	import { upsAPI } from '$lib/api';
	import type { UPSEntry } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field';
	import * as Table from '$lib/components/ui/table';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { Input } from '$lib/components/ui/input';
	import { toast } from 'svelte-sonner';
	import { superForm, defaults } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { upsSchema } from '$lib/schemas';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Zap from '@lucide/svelte/icons/zap';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';

	let upsList = $state<UPSEntry[]>([]);
	let loading = $state(true);
	let addDialogOpen = $state(false);
	let editDialogOpen = $state(false);
	let deletingId = $state<string | null>(null);
	let editingId = $state<string | null>(null);

	// Superform for Add dialog
	const addForm = superForm(defaults({ name: '', host: '', ups_name: 'ups' }, zod4(upsSchema)), {
		SPA: true,
		validators: zod4(upsSchema),
		onUpdate: async ({ form }) => {
			if (!form.valid) return;

			const res = await upsAPI.create({
				name: form.data.name,
				host: form.data.host,
				ups_name: form.data.ups_name
			});

			if (res.success && res.data) {
				upsList = [...upsList, res.data];
				toast.success(`UPS "${form.data.name}" added`);
				addDialogOpen = false;
				addForm.reset();
			} else {
				toast.error(res.error || 'Failed to add UPS');
			}
		}
	});

	// Superform for Edit dialog
	const editForm = superForm(defaults({ name: '', host: '', ups_name: 'ups' }, zod4(upsSchema)), {
		SPA: true,
		validators: zod4(upsSchema),
		onUpdate: async ({ form }) => {
			if (!form.valid || !editingId) return;

			const res = await upsAPI.update(editingId, {
				name: form.data.name,
				host: form.data.host,
				ups_name: form.data.ups_name
			});

			if (res.success) {
				await loadUPS();
				toast.success(`UPS "${form.data.name}" updated`);
				editDialogOpen = false;
				editForm.reset();
				editingId = null;
			} else {
				toast.error(res.error || 'Failed to update UPS');
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

	async function loadUPS() {
		const res = await upsAPI.getAll();
		if (res.success && res.data) {
			upsList = res.data;
		}
		loading = false;
	}

	function openAddDialog() {
		addForm.reset();
		addDialogOpen = true;
	}

	function openEditDialog(ups: UPSEntry) {
		editingId = ups.id;
		editForm.reset({
			data: {
				name: ups.name,
				host: ups.host,
				ups_name: ups.ups_name
			}
		});
		editDialogOpen = true;
	}

	async function deleteUPS(ups: UPSEntry) {
		deletingId = ups.id;
		const res = await upsAPI.delete(ups.id);

		if (res.success) {
			upsList = upsList.filter((u) => u.id !== ups.id);
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
			<Plus class="h-4 w-4" />
			Add UPS
		</Button>
	</div>

	<!-- Add Dialog -->
	<Dialog.Root bind:open={addDialogOpen}>
		<Dialog.Content class="sm:max-w-md">
			<Dialog.Header>
				<Dialog.Title>Add UPS Connection</Dialog.Title>
				<Dialog.Description>Connect to a NUT (Network UPS Tools) server</Dialog.Description>
			</Dialog.Header>
			<form method="POST" use:addEnhance class="space-y-4">
				<Field.Field data-invalid={$addErrors.name ? true : undefined}>
					<Field.Label for="add-name">Display Name</Field.Label>
					<Input id="add-name" placeholder="e.g., Office UPS" bind:value={$addFormData.name} />
					{#if $addErrors.name}
						<Field.Error>{$addErrors.name}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field data-invalid={$addErrors.host ? true : undefined}>
					<Field.Label for="add-host">NUT Server Host</Field.Label>
					<Input
						id="add-host"
						placeholder="localhost:3493"
						bind:value={$addFormData.host}
						class="font-mono"
					/>
					<Field.Description>Host and port of the NUT server (default port: 3493)</Field.Description>
					{#if $addErrors.host}
						<Field.Error>{$addErrors.host}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field data-invalid={$addErrors.ups_name ? true : undefined}>
					<Field.Label for="add-ups-name">UPS Name</Field.Label>
					<Input
						id="add-ups-name"
						placeholder="ups"
						bind:value={$addFormData.ups_name}
						class="font-mono"
					/>
					<Field.Description>UPS identifier as configured in NUT (usually "ups")</Field.Description>
					{#if $addErrors.ups_name}
						<Field.Error>{$addErrors.ups_name}</Field.Error>
					{/if}
				</Field.Field>
				<Dialog.Footer>
					<Button type="button" variant="outline" onclick={() => (addDialogOpen = false)}>
						Cancel
					</Button>
					<Button type="submit" disabled={$addSubmitting}>
						{#if $addSubmitting}
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
				<Dialog.Description>Update UPS connection information</Dialog.Description>
			</Dialog.Header>
			<form method="POST" use:editEnhance class="space-y-4">
				<Field.Field data-invalid={$editErrors.name ? true : undefined}>
					<Field.Label for="edit-name">Display Name</Field.Label>
					<Input id="edit-name" placeholder="e.g., Office UPS" bind:value={$editFormData.name} />
					{#if $editErrors.name}
						<Field.Error>{$editErrors.name}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field data-invalid={$editErrors.host ? true : undefined}>
					<Field.Label for="edit-host">NUT Server Host</Field.Label>
					<Input
						id="edit-host"
						placeholder="localhost:3493"
						bind:value={$editFormData.host}
						class="font-mono"
					/>
					<Field.Description>Host and port of the NUT server (default port: 3493)</Field.Description>
					{#if $editErrors.host}
						<Field.Error>{$editErrors.host}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field data-invalid={$editErrors.ups_name ? true : undefined}>
					<Field.Label for="edit-ups-name">UPS Name</Field.Label>
					<Input
						id="edit-ups-name"
						placeholder="ups"
						bind:value={$editFormData.ups_name}
						class="font-mono"
					/>
					<Field.Description>UPS identifier as configured in NUT (usually "ups")</Field.Description>
					{#if $editErrors.ups_name}
						<Field.Error>{$editErrors.ups_name}</Field.Error>
					{/if}
				</Field.Field>
				<Dialog.Footer>
					<Button type="button" variant="outline" onclick={() => (editDialogOpen = false)}>
						Cancel
					</Button>
					<Button type="submit" disabled={$editSubmitting}>
						{#if $editSubmitting}
							<RefreshCw class="h-4 w-4 mr-1 animate-spin" />
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
						<Table.Head>Name</Table.Head>
						<Table.Head>Host</Table.Head>
						<Table.Head>UPS Name</Table.Head>
						<Table.Head class="w-24 text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each [1, 2] as _}
						<Table.Row>
							<Table.Cell><div class="h-4 w-24 bg-muted rounded animate-pulse"></div></Table.Cell>
							<Table.Cell><div class="h-4 w-32 bg-muted rounded animate-pulse"></div></Table.Cell>
							<Table.Cell><div class="h-4 w-16 bg-muted rounded animate-pulse"></div></Table.Cell>
							<Table.Cell><div class="h-8 w-20 bg-muted rounded animate-pulse ml-auto"></div></Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
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
		<div class="rounded-md border">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>Name</Table.Head>
						<Table.Head>Host</Table.Head>
						<Table.Head>UPS Name</Table.Head>
						<Table.Head class="w-24 text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each upsList as ups}
						<Table.Row>
							<Table.Cell class="font-medium">{ups.name}</Table.Cell>
							<Table.Cell class="font-mono text-muted-foreground">{ups.host}</Table.Cell>
							<Table.Cell class="font-mono text-muted-foreground">{ups.ups_name}</Table.Cell>
							<Table.Cell class="text-right">
								<div class="flex justify-end gap-1">
									<Tooltip.Root>
										<Tooltip.Trigger>
											<Button size="icon" variant="outline" onclick={() => openEditDialog(ups)}>
												<Pencil class="h-4 w-4" />
											</Button>
										</Tooltip.Trigger>
										<Tooltip.Content>
											<p>Edit UPS</p>
										</Tooltip.Content>
									</Tooltip.Root>
									<Tooltip.Root>
										<Tooltip.Trigger>
											<Button
												size="icon"
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
										</Tooltip.Trigger>
										<Tooltip.Content>
											<p>Delete UPS</p>
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
