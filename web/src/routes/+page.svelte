<script lang="ts">
	import { deviceAPI, upsAPI } from '$lib/api';
	import type { DeviceStatus, UPSStatus } from '$lib/api';
	import {
		formatRuntime,
		formatWattage,
		deviceStatusBadgeClass,
		deviceStatusLabel
	} from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { Badge } from '$lib/components/ui/badge';
	import { toast } from 'svelte-sonner';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Power from '@lucide/svelte/icons/power';
	import Plug from '@lucide/svelte/icons/plug';
	import Battery from '@lucide/svelte/icons/battery';
	import BatteryCharging from '@lucide/svelte/icons/battery-charging';
	import BatteryWarning from '@lucide/svelte/icons/battery-warning';
	import WifiOff from '@lucide/svelte/icons/wifi-off';
	import Zap from '@lucide/svelte/icons/zap';

	let wakingDevice = $state<string | null>(null);

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

	// UPS status query with auto-refresh
	const upsQuery = createQuery(() => ({
		queryKey: ['ups', 'status'],
		queryFn: async () => {
			const res = await upsAPI.getStatus();
			if (res.success && res.data) {
				return res.data;
			}
			throw new Error(res.error || 'Failed to fetch UPS status');
		},
		// Match the backend's 15s UPS cache refresh — polling slower than the
		// cache updates lets the dashboard lag real state by up to ~45s.
		refetchInterval: 15000,
		refetchIntervalInBackground: false
	}));

	// Derived state - separate loading states
	const devices = $derived(devicesQuery.data ?? []);
	const upsStatuses = $derived(upsQuery.data ?? []);
	const upsLoading = $derived(upsQuery.isLoading);
	const devicesLoading = $derived(devicesQuery.isLoading);
	const upsRefreshing = $derived(upsQuery.isFetching);
	const devicesRefreshing = $derived(devicesQuery.isFetching);
	const upsLastUpdated = $derived(
		upsQuery.dataUpdatedAt ? new Date(upsQuery.dataUpdatedAt) : null
	);
	const devicesLastUpdated = $derived(
		devicesQuery.dataUpdatedAt ? new Date(devicesQuery.dataUpdatedAt) : null
	);

	async function refreshUPS() {
		await queryClient.invalidateQueries({ queryKey: ['ups', 'status'] });
	}

	async function refreshDevices() {
		await queryClient.invalidateQueries({ queryKey: ['devices', 'status'] });
	}

	async function wakeDevice(device: DeviceStatus) {
		wakingDevice = device.id;
		const res = await deviceAPI.wake(device.id);
		if (res.success) {
			toast.success(`Wake packet sent to ${device.name}`);
		} else {
			toast.error(res.error || 'Failed to wake device');
		}
		wakingDevice = null;
	}

	// Hero status for the UPS card: one short statement, an icon, and a sub-line
	// of supporting facts.  Designed for the 10-second visit — the user opens
	// the dashboard after a power blip, sees "On grid power" in emerald, closes
	// the tab.  All five states map to a single component shape so we keep
	// hierarchy stable as state changes.
	type Tone = 'healthy' | 'warning' | 'critical' | 'unknown';
	type Headline = {
		text: string;
		tone: Tone;
		sub: string;
		icon: typeof Plug;
		// When present, renders a thin inline load bar after the sub-line.
		// Only set on healthy states — on battery / critical, the focal info
		// is runtime not load, and a bar would be visual noise.
		load?: number;
	};

	function headline(ups: UPSStatus): Headline {
		if (!ups.online) {
			return {
				text: 'Unreachable',
				tone: 'critical',
				sub: ups.error || `Can't reach ${ups.host}`,
				icon: WifiOff
			};
		}
		if (ups.is_low_battery) {
			return {
				text: 'Battery critically low',
				tone: 'critical',
				sub:
					ups.battery_runtime > 0
						? `${formatRuntime(ups.battery_runtime)} until shutdown · ${ups.battery_charge}% remaining`
						: `${ups.battery_charge}% remaining — shutting down soon`,
				icon: BatteryWarning
			};
		}
		if (ups.is_on_battery || ups.is_discharging) {
			return {
				text: 'On battery',
				tone: 'warning',
				sub:
					ups.battery_runtime > 0
						? `${formatRuntime(ups.battery_runtime)} runtime · ${ups.battery_charge}% remaining`
						: `${ups.battery_charge}% remaining`,
				icon: Battery
			};
		}
		if (ups.is_charging) {
			return {
				text: 'On grid · charging',
				tone: 'healthy',
				sub: `${ups.battery_charge}% charge · ${ups.load}% load`,
				icon: BatteryCharging,
				load: ups.load
			};
		}
		return {
			text: 'On grid power',
			tone: 'healthy',
			sub: `${ups.battery_charge}% charge · ${ups.load}% load`,
			icon: Plug,
			load: ups.load
		};
	}

	// Threshold colors match the semantic palette in app.css.
	function loadBarColor(load: number): string {
		if (load < 50) return 'bg-emerald-500';
		if (load < 80) return 'bg-amber-500';
		return 'bg-red-500';
	}

	function toneText(tone: Tone): string {
		switch (tone) {
			case 'healthy':
				return 'text-emerald-500';
			case 'warning':
				return 'text-amber-500';
			case 'critical':
				return 'text-red-500';
			default:
				return 'text-muted-foreground';
		}
	}
</script>

<div class="space-y-10">
	<!-- UPS Section -->
	<section>
		<!-- Overline-style section header.  Cards below are the visual focal,
		     so the section heading stays quiet and organizational. -->
		<div class="mb-3 flex items-baseline justify-between gap-3">
			<h2 class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
				UPS
			</h2>
			<div class="flex items-center gap-3 text-xs text-muted-foreground">
				{#if upsLastUpdated}
					<span class="tabular-nums">Updated {upsLastUpdated.toLocaleTimeString()}</span>
				{/if}
				<Button
					variant="ghost"
					size="icon"
					class="h-7 w-7 text-muted-foreground hover:text-foreground"
					onclick={refreshUPS}
					disabled={upsRefreshing}
					aria-label="Refresh UPS status"
				>
					<RefreshCw class="h-3.5 w-3.5 {upsRefreshing ? 'animate-spin' : ''}" />
				</Button>
			</div>
		</div>

		{#if upsLoading}
			<!-- Skeleton matches the loaded card's shape (identifier line +
			     hero/diagnostics row) so the layout doesn't jump on data arrival. -->
			<div class="space-y-4">
				{#each [1, 2] as _}
					<Card.Root>
						<Card.Content class="p-6">
							<div class="animate-pulse space-y-5">
								<div class="flex items-baseline justify-between gap-3">
									<div class="h-4 w-24 rounded bg-muted"></div>
									<div class="h-3 w-32 rounded bg-muted"></div>
								</div>
								<div class="flex flex-col gap-6 md:flex-row md:items-center">
									<div class="flex flex-1 items-center gap-4">
										<div class="h-10 w-10 shrink-0 rounded bg-muted"></div>
										<div class="flex-1 space-y-2">
											<div class="h-7 w-2/3 rounded bg-muted"></div>
											<div class="h-3 w-3/4 rounded bg-muted"></div>
										</div>
									</div>
									<div class="grid flex-1 grid-cols-2 gap-x-6 gap-y-3">
										<div class="space-y-1.5">
											<div class="h-2.5 w-12 rounded bg-muted"></div>
											<div class="h-3.5 w-16 rounded bg-muted"></div>
										</div>
										<div class="space-y-1.5">
											<div class="h-2.5 w-12 rounded bg-muted"></div>
											<div class="h-3.5 w-16 rounded bg-muted"></div>
										</div>
										<div class="space-y-1.5">
											<div class="h-2.5 w-12 rounded bg-muted"></div>
											<div class="h-3.5 w-16 rounded bg-muted"></div>
										</div>
										<div class="space-y-1.5">
											<div class="h-2.5 w-12 rounded bg-muted"></div>
											<div class="h-3.5 w-16 rounded bg-muted"></div>
										</div>
									</div>
								</div>
							</div>
						</Card.Content>
					</Card.Root>
				{/each}
			</div>
		{:else if upsStatuses.length === 0}
			<Card.Root>
				<Card.Content class="p-6 text-center text-muted-foreground">
					<Zap class="mx-auto mb-4 h-12 w-12 opacity-50" />
					<p>No UPS connections configured.</p>
					<Button variant="link" href="/ups" class="mt-2">Add UPS Connection</Button>
				</Card.Content>
			</Card.Root>
		{:else}
			<!-- Vertical stack of full-width cards.  Inside each card, the hero
			     status (left) and diagnostics (right) sit side-by-side at md+
			     with a vertical divider — asymmetric horizontal flow instead of
			     a stack of identical mirrored panels. -->
			<div class="space-y-4">
				{#each upsStatuses as ups (ups.id)}
					{@const h = headline(ups)}
					{@const HIcon = h.icon}
					<!-- Override Card.Root's default py-6 + gap-6 so our identifier /
					     hero / diagnostics / footer zones butt up against each other
					     with explicit borders, instead of inheriting 24px gaps. -->
					<Card.Root class="gap-0 py-0">
						<!-- Header zone: identifier line + optional manufacturer subtitle.
						     Bottom border separates it from the hero/diagnostics zone. -->
						<div class="border-b px-5 pb-3 pt-4">
							<div class="flex items-baseline justify-between gap-3">
								<Card.Title class="text-base font-semibold tracking-tight">
									{ups.name}
								</Card.Title>
								{#if ups.host}
									<span class="font-mono text-xs text-muted-foreground tabular-nums">
										{ups.host}
									</span>
								{/if}
							</div>
							{#if ups.model || ups.manufacturer}
								<p class="pt-0.5 text-xs text-muted-foreground">
									{[ups.manufacturer, ups.model].filter(Boolean).join(' ')}
								</p>
							{/if}
						</div>

						<!-- Hero + diagnostics zone.  Stacked on mobile, side-by-side
						     at md+ with a vertical divider between them. -->
						<div class="flex flex-col md:flex-row md:items-stretch">
							<!-- Hero status -->
							<div class="flex flex-1 items-start gap-3 px-5 py-5">
								<HIcon class="h-9 w-9 shrink-0 {toneText(h.tone)}" strokeWidth={1.75} />
								<div class="min-w-0 flex-1">
									<p class="text-3xl font-semibold leading-tight tracking-tight {toneText(h.tone)}">
										{h.text}
									</p>
									{#if h.sub}
										<div class="mt-1 flex flex-wrap items-center gap-x-2 text-sm text-muted-foreground tabular-nums">
											<span>{h.sub}</span>
											{#if h.load !== undefined}
												<span
													class="inline-block h-1.5 w-14 overflow-hidden rounded-full bg-muted"
													aria-hidden="true"
												>
													<span
														class="block h-full rounded-full {loadBarColor(h.load)}"
														style="width: {Math.max(0, Math.min(100, h.load))}%"
													></span>
												</span>
											{/if}
										</div>
									{/if}
								</div>
							</div>

							<!-- Diagnostics: 2x2 grid, sits to the right of hero on md+. -->
							{#if !ups.error && ups.online}
								<dl
									class="grid flex-1 grid-cols-2 gap-x-6 gap-y-2.5 border-t px-5 py-4 text-sm md:border-l md:border-t-0 md:py-5"
								>
									<div>
										<dt class="text-xs text-muted-foreground">Power</dt>
										<dd class="font-medium tabular-nums">
											{#if ups.estimated_wattage > 0}
												{formatWattage(ups.estimated_wattage)}
											{:else if ups.power > 0}
												{formatWattage(ups.power)}
											{:else}
												—
											{/if}
											{#if ups.nominal > 0}
												<span class="text-muted-foreground">
													/ {formatWattage(ups.nominal)}
												</span>
											{/if}
										</dd>
									</div>
									<div>
										<dt class="text-xs text-muted-foreground">Runtime</dt>
										<dd class="font-medium tabular-nums">
											{formatRuntime(ups.battery_runtime)}
										</dd>
									</div>
									<div>
										<dt class="text-xs text-muted-foreground">Input</dt>
										<dd class="font-medium tabular-nums">
											{ups.input_voltage > 0 ? `${ups.input_voltage.toFixed(0)}V` : '—'}
											{#if ups.input_frequency > 0}
												<span class="text-muted-foreground">
													{ups.input_frequency.toFixed(1)}Hz
												</span>
											{/if}
										</dd>
									</div>
									<div>
										<dt class="text-xs text-muted-foreground">Output</dt>
										<dd class="font-medium tabular-nums">
											{ups.output_voltage > 0 ? `${ups.output_voltage.toFixed(0)}V` : '—'}
											{#if ups.output_frequency > 0}
												<span class="text-muted-foreground">
													{ups.output_frequency.toFixed(1)}Hz
												</span>
											{/if}
										</dd>
									</div>
								</dl>
							{/if}
						</div>

						<!-- Status footer: humanized label first, then the raw NUT
						     code in mono ("things you'd grep").  Skip the label when
						     it's identical to the raw code (no tokens matched) so it
						     isn't shown twice. -->
						{#if !ups.error && ups.status}
							<p class="flex flex-wrap items-center gap-x-2 border-t px-5 py-2 text-xs text-muted-foreground">
								{#if ups.status_label && ups.status_label !== ups.status}
									<span>{ups.status_label}</span>
								{/if}
								<span class="font-mono">{ups.status}</span>
							</p>
						{/if}
					</Card.Root>
				{/each}
			</div>
		{/if}
	</section>

	<!-- Devices Section -->
	<section>
		<div class="mb-3 flex items-baseline justify-between gap-3">
			<h2 class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
				Devices
			</h2>
			<div class="flex items-center gap-3 text-xs text-muted-foreground">
				{#if devicesLastUpdated}
					<span class="tabular-nums">Updated {devicesLastUpdated.toLocaleTimeString()}</span>
				{/if}
				<Button
					variant="ghost"
					size="icon"
					class="h-7 w-7 text-muted-foreground hover:text-foreground"
					onclick={refreshDevices}
					disabled={devicesRefreshing}
					aria-label="Refresh device status"
				>
					<RefreshCw class="h-3.5 w-3.5 {devicesRefreshing ? 'animate-spin' : ''}" />
				</Button>
			</div>
		</div>

		{#if devicesLoading}
			<Card.Root>
				<Card.Content class="p-6">
					<div class="animate-pulse space-y-3">
						<div class="h-4 bg-muted rounded w-full"></div>
						<div class="h-4 bg-muted rounded w-full"></div>
						<div class="h-4 bg-muted rounded w-full"></div>
					</div>
				</Card.Content>
			</Card.Root>
		{:else if devices.length === 0}
			<Card.Root>
				<Card.Content class="p-6 text-center text-muted-foreground">
					<Power class="h-12 w-12 mx-auto mb-4 opacity-50" />
					<p>No devices configured.</p>
					<Button variant="link" href="/devices" class="mt-2">Add Device</Button>
				</Card.Content>
			</Card.Root>
		{:else}
			<div class="rounded-lg border">
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head class="w-24">Status</Table.Head>
							<Table.Head>Name</Table.Head>
							<Table.Head>MAC Address</Table.Head>
							<Table.Head>IP Address</Table.Head>
							<Table.Head class="w-16 text-right">Action</Table.Head>
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
								<Table.Cell class="font-mono text-sm text-muted-foreground">{device.mac}</Table.Cell>
								<Table.Cell class="font-mono text-sm text-muted-foreground">
									{device.ip || '-'}
								</Table.Cell>
								<Table.Cell class="text-right">
									<Tooltip.Root>
										<Tooltip.Trigger>
											<Button
												size="icon"
												variant="outline"
												onclick={() => wakeDevice(device)}
												disabled={wakingDevice === device.id}
												aria-label="Wake {device.name}"
											>
												{#if wakingDevice === device.id}
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
								</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			</div>
		{/if}
	</section>
</div>
