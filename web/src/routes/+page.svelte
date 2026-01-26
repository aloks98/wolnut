<script lang="ts">
	import { deviceAPI, upsAPI } from '$lib/api';
	import type { DeviceStatus, UPSStatus } from '$lib/api';
	import {
		formatRuntime,
		formatWattage,
		getBatteryColor,
		getBatteryTextColor,
		getLoadColor
	} from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { Badge } from '$lib/components/ui/badge';
	import { toast } from 'svelte-sonner';
	import { PieChart } from 'layerchart';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Power from '@lucide/svelte/icons/power';
	import Battery from '@lucide/svelte/icons/battery';
	import BatteryCharging from '@lucide/svelte/icons/battery-charging';
	import BatteryWarning from '@lucide/svelte/icons/battery-warning';
	import Zap from '@lucide/svelte/icons/zap';
	import Clock from '@lucide/svelte/icons/clock';
	import Gauge from '@lucide/svelte/icons/gauge';
	import Thermometer from '@lucide/svelte/icons/thermometer';
	import Activity from '@lucide/svelte/icons/activity';
	import Info from '@lucide/svelte/icons/info';

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
		refetchInterval: 30000,
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

	function getStatusBadgeClass(ups: UPSStatus): string {
		if (!ups.online) return 'bg-red-500/15 text-red-500 border-red-500/20';
		if (ups.is_low_battery) return 'bg-red-500/15 text-red-500 border-red-500/20';
		if (ups.is_on_battery || ups.is_discharging)
			return 'bg-amber-500/15 text-amber-500 border-amber-500/20';
		if (ups.is_charging) return 'bg-blue-500/15 text-blue-500 border-blue-500/20';
		return 'bg-emerald-500/15 text-emerald-500 border-emerald-500/20';
	}
</script>

<div class="space-y-8">
	<!-- UPS Section -->
	<section>
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-xl font-semibold">UPS Status</h2>
			<div class="flex items-center gap-2">
				{#if upsLastUpdated}
					<span class="text-xs text-muted-foreground">
						Updated {upsLastUpdated.toLocaleTimeString()}
					</span>
				{/if}
				<Button variant="outline" size="sm" onclick={refreshUPS} disabled={upsRefreshing}>
					<RefreshCw class="h-4 w-4 mr-1 {upsRefreshing ? 'animate-spin' : ''}" />
					Refresh
				</Button>
			</div>
		</div>

		{#if upsLoading}
			<div class="grid gap-6 lg:grid-cols-2">
				{#each [1, 2] as _}
					<Card.Root>
						<Card.Content class="p-6">
							<div class="animate-pulse space-y-4">
								<div class="h-4 bg-muted rounded w-1/2"></div>
								<div class="h-40 bg-muted rounded"></div>
							</div>
						</Card.Content>
					</Card.Root>
				{/each}
			</div>
		{:else if upsStatuses.length === 0}
			<Card.Root>
				<Card.Content class="p-6 text-center text-muted-foreground">
					<Zap class="h-12 w-12 mx-auto mb-4 opacity-50" />
					<p>No UPS connections configured.</p>
					<Button variant="link" href="/ups" class="mt-2">Add UPS Connection</Button>
				</Card.Content>
			</Card.Root>
		{:else}
			<div class="grid gap-6 lg:grid-cols-2">
				{#each upsStatuses as ups}
					<Card.Root class="overflow-hidden">
						<Card.Header class="pb-3">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2">
									<Card.Title class="text-lg">{ups.name}</Card.Title>
									{#if ups.is_charging}
										<BatteryCharging class="h-5 w-5 text-emerald-500" />
									{:else if ups.is_on_battery || ups.is_discharging}
										<BatteryWarning class="h-5 w-5 text-amber-500" />
									{/if}
								</div>
								<Badge variant="outline" class={getStatusBadgeClass(ups)}>
									{ups.status_label || ups.status || 'Unknown'}
								</Badge>
							</div>
							{#if ups.model || ups.manufacturer}
								<p class="text-sm text-muted-foreground">
									{[ups.manufacturer, ups.model].filter(Boolean).join(' ')}
								</p>
							{/if}
						</Card.Header>
						<Card.Content class="pt-2">
							{#if ups.error}
								<p class="text-sm text-destructive">{ups.error}</p>
							{:else}
								<div class="flex justify-around py-4">
									<!-- Battery Donut Chart -->
									<div class="flex flex-col items-center">
										<div class="relative w-32 h-32">
											<PieChart
												data={[{ value: ups.battery_charge }]}
												value="value"
												innerRadius={0.7}
												cornerRadius={4}
												padAngle={0.02}
												series={[
													{
														key: 'charge',
														value: 'value',
														maxValue: 100,
														color: getBatteryColor(ups.battery_charge)
													}
												]}
											/>
											<div
												class="absolute inset-0 flex flex-col items-center justify-center"
											>
												<span
													class="text-2xl font-bold {getBatteryTextColor(ups.battery_charge)}"
												>
													{ups.battery_charge}%
												</span>
											</div>
										</div>
										<span class="text-sm text-muted-foreground mt-2">Battery</span>
									</div>

									<!-- Load Donut Chart -->
									<div class="flex flex-col items-center">
										<div class="relative w-32 h-32">
											<PieChart
												data={[{ value: ups.load }]}
												value="value"
												innerRadius={0.7}
												cornerRadius={4}
												padAngle={0.02}
												series={[
													{
														key: 'load',
														value: 'value',
														maxValue: 100,
														color: getLoadColor(ups.load)
													}
												]}
											/>
											<div
												class="absolute inset-0 flex flex-col items-center justify-center"
											>
												<span class="text-2xl font-bold">
													{ups.load}%
												</span>
											</div>
										</div>
										<span class="text-sm text-muted-foreground mt-2">Load</span>
									</div>
								</div>

								<!-- Stats Grid -->
								<div class="grid grid-cols-2 gap-4 mt-6 pt-5 border-t">
									<!-- Runtime -->
									<div class="flex items-center gap-3">
										<Clock class="h-5 w-5 text-muted-foreground" />
										<div>
											<p class="text-xs text-muted-foreground">Runtime</p>
											<p class="text-base font-medium">
												{formatRuntime(ups.battery_runtime)}
											</p>
										</div>
									</div>

									<!-- Power Draw -->
									<div class="flex items-center gap-3">
										<Activity class="h-5 w-5 text-muted-foreground" />
										<div>
											<p class="text-xs text-muted-foreground">Power</p>
											<p class="text-base font-medium">
												{#if ups.estimated_wattage > 0}
													{formatWattage(ups.estimated_wattage)}
												{:else if ups.power > 0}
													{formatWattage(ups.power)}
												{:else}
													--
												{/if}
												{#if ups.nominal > 0}
													<span class="text-sm text-muted-foreground"
														>/ {formatWattage(ups.nominal)}</span
													>
												{/if}
											</p>
										</div>
									</div>

									<!-- Input Voltage -->
									<div class="flex items-center gap-3">
										<Zap class="h-5 w-5 text-muted-foreground" />
										<div>
											<p class="text-xs text-muted-foreground">Input</p>
											<p class="text-base font-medium">
												{ups.input_voltage > 0
													? `${ups.input_voltage.toFixed(0)}V`
													: '--'}
												{#if ups.input_frequency > 0}
													<span class="text-sm text-muted-foreground"
														>{ups.input_frequency.toFixed(1)}Hz</span
													>
												{/if}
											</p>
										</div>
									</div>

									<!-- Output Voltage -->
									<div class="flex items-center gap-3">
										<Gauge class="h-5 w-5 text-muted-foreground" />
										<div>
											<p class="text-xs text-muted-foreground">Output</p>
											<p class="text-base font-medium">
												{ups.output_voltage > 0
													? `${ups.output_voltage.toFixed(0)}V`
													: '--'}
												{#if ups.output_frequency > 0}
													<span class="text-sm text-muted-foreground"
														>{ups.output_frequency.toFixed(1)}Hz</span
													>
												{/if}
											</p>
										</div>
									</div>

									<!-- Battery Voltage -->
									{#if ups.battery_voltage > 0}
										<div class="flex items-center gap-3">
											<Battery class="h-5 w-5 text-muted-foreground" />
											<div>
												<p class="text-xs text-muted-foreground">Batt. Voltage</p>
												<p class="text-base font-medium">
													{ups.battery_voltage.toFixed(1)}V
												</p>
											</div>
										</div>
									{/if}

									<!-- Temperature -->
									{#if ups.temperature > 0}
										<div class="flex items-center gap-3">
											<Thermometer class="h-5 w-5 text-muted-foreground" />
											<div>
												<p class="text-xs text-muted-foreground">Temperature</p>
												<p class="text-base font-medium">
													{ups.temperature.toFixed(0)}°C
												</p>
											</div>
										</div>
									{/if}

									<!-- Current -->
									{#if ups.current > 0}
										<div class="flex items-center gap-3">
											<Activity class="h-5 w-5 text-muted-foreground" />
											<div>
												<p class="text-xs text-muted-foreground">Current</p>
												<p class="text-base font-medium">{ups.current.toFixed(1)}A</p>
											</div>
										</div>
									{/if}
								</div>

								<!-- Raw Status -->
								<div class="mt-4 pt-4 border-t">
									<div class="flex items-center gap-2 text-sm text-muted-foreground">
										<Info class="h-4 w-4" />
										<span>Raw: {ups.status}</span>
										{#if ups.host}
											<span class="ml-auto">{ups.host}</span>
										{/if}
									</div>
								</div>
							{/if}
						</Card.Content>
					</Card.Root>
				{/each}
			</div>
		{/if}
	</section>

	<!-- Devices Section -->
	<section>
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-xl font-semibold">Wake-on-LAN Devices</h2>
			<div class="flex items-center gap-2">
				{#if devicesLastUpdated}
					<span class="text-xs text-muted-foreground">
						Updated {devicesLastUpdated.toLocaleTimeString()}
					</span>
				{/if}
				<Button variant="outline" size="sm" onclick={refreshDevices} disabled={devicesRefreshing}>
					<RefreshCw class="h-4 w-4 mr-1 {devicesRefreshing ? 'animate-spin' : ''}" />
					Refresh
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
						{#each devices as device}
							<Table.Row>
								<Table.Cell>
									<Badge
										variant="outline"
										class={device.online === null
											? 'bg-muted/50 text-muted-foreground border-muted'
											: device.online
												? 'bg-emerald-500/15 text-emerald-500 border-emerald-500/20'
												: 'bg-red-500/15 text-red-500 border-red-500/20'}
									>
										{device.online === null ? 'Unknown' : device.online ? 'Online' : 'Offline'}
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
