<script lang="ts">
	import { onMount } from 'svelte';
	import { deviceAPI, upsAPI } from '$lib/api';
	import type { Device, UPSStatus } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { toast } from 'svelte-sonner';
	import { PieChart } from 'layerchart';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Power from '@lucide/svelte/icons/power';
	import Battery from '@lucide/svelte/icons/battery';
	import BatteryCharging from '@lucide/svelte/icons/battery-charging';
	import Zap from '@lucide/svelte/icons/zap';
	import Clock from '@lucide/svelte/icons/clock';
	import Gauge from '@lucide/svelte/icons/gauge';
	import Thermometer from '@lucide/svelte/icons/thermometer';
	import Activity from '@lucide/svelte/icons/activity';
	import Info from '@lucide/svelte/icons/info';

	let devices = $state<Device[]>([]);
	let upsStatuses = $state<UPSStatus[]>([]);
	let loading = $state(true);
	let refreshing = $state(false);
	let wakingDevice = $state<string | null>(null);
	let lastUpdated = $state<Date | null>(null);

	async function loadData() {
		const [devicesRes, upsRes] = await Promise.all([
			deviceAPI.getAll(),
			upsAPI.getStatus()
		]);

		if (devicesRes.success && devicesRes.data) {
			devices = devicesRes.data;
		}
		if (upsRes.success && upsRes.data) {
			upsStatuses = upsRes.data;
		}
		lastUpdated = new Date();
		loading = false;
	}

	async function refresh() {
		refreshing = true;
		await loadData();
		refreshing = false;
	}

	async function wakeDevice(device: Device) {
		wakingDevice = device.id;
		const res = await deviceAPI.wake(device.id);
		if (res.success) {
			toast.success(`Wake packet sent to ${device.name}`);
		} else {
			toast.error(res.error || 'Failed to wake device');
		}
		wakingDevice = null;
	}

	function formatRuntime(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m`;
		const hours = Math.floor(minutes / 60);
		const mins = minutes % 60;
		return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
	}

	function getStatusVariant(ups: UPSStatus): 'default' | 'secondary' | 'destructive' | 'outline' {
		if (!ups.online) return 'destructive';
		if (ups.is_on_battery) return 'secondary';
		if (ups.is_low_battery) return 'destructive';
		return 'default';
	}

	function getBatteryColor(charge: number): string {
		if (charge > 50) return '#22c55e'; // green-500
		if (charge > 20) return '#f59e0b'; // amber-500
		return '#ef4444'; // red-500
	}

	function getBatteryTextColor(charge: number): string {
		if (charge > 50) return 'text-emerald-500';
		if (charge > 20) return 'text-amber-500';
		return 'text-red-500';
	}

	function getLoadColor(load: number): string {
		if (load < 50) return '#22c55e'; // green
		if (load < 80) return '#f59e0b'; // amber
		return '#ef4444'; // red
	}

	function formatWattage(watts: number): string {
		if (watts >= 1000) {
			return `${(watts / 1000).toFixed(1)}kW`;
		}
		return `${Math.round(watts)}W`;
	}

	onMount(() => {
		loadData();

		// Auto-refresh every 30 seconds
		const interval = setInterval(() => {
			if (!document.hidden) {
				loadData();
			}
		}, 30000);

		return () => clearInterval(interval);
	});
</script>

<div class="space-y-8">
	<!-- UPS Section -->
	<section>
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-xl font-semibold">UPS Status</h2>
			<div class="flex items-center gap-2">
				{#if lastUpdated}
					<span class="text-xs text-muted-foreground">
						Updated {lastUpdated.toLocaleTimeString()}
					</span>
				{/if}
				<Button variant="outline" size="sm" onclick={refresh} disabled={refreshing}>
					<RefreshCw class="h-4 w-4 mr-1 {refreshing ? 'animate-spin' : ''}" />
					Refresh
				</Button>
			</div>
		</div>

		{#if loading}
			<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
				{#each [1, 2] as _}
					<Card.Root>
						<Card.Content class="p-6">
							<div class="animate-pulse space-y-4">
								<div class="h-4 bg-muted rounded w-1/2"></div>
								<div class="h-32 bg-muted rounded"></div>
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
			<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
				{#each upsStatuses as ups}
					<Card.Root class="overflow-hidden">
						<Card.Header class="pb-2">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2">
									<Card.Title class="text-base">{ups.name}</Card.Title>
									{#if ups.is_charging}
										<BatteryCharging class="h-4 w-4 text-amber-500" />
									{/if}
								</div>
								<Badge variant={getStatusVariant(ups)}>
									{ups.status_label || ups.status || 'Unknown'}
								</Badge>
							</div>
							{#if ups.model || ups.manufacturer}
								<p class="text-xs text-muted-foreground">
									{[ups.manufacturer, ups.model].filter(Boolean).join(' ')}
								</p>
							{/if}
						</Card.Header>
						<Card.Content>
							{#if ups.error}
								<p class="text-sm text-destructive">{ups.error}</p>
							{:else}
								<div class="grid grid-cols-3 gap-2">
									<!-- Battery Donut Chart -->
									<div class="flex flex-col items-center">
										<div class="relative w-20 h-20">
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
											<div class="absolute inset-0 flex flex-col items-center justify-center">
												<span class="text-base font-bold {getBatteryTextColor(ups.battery_charge)}">
													{ups.battery_charge}%
												</span>
											</div>
										</div>
										<span class="text-xs text-muted-foreground mt-1">Battery</span>
									</div>

									<!-- Runtime Display -->
									<div class="flex flex-col items-center justify-center">
										<div class="flex flex-col items-center justify-center w-20 h-20 rounded-full border-2 border-muted bg-muted/30">
											<Clock class="h-4 w-4 text-muted-foreground mb-1" />
											<span class="text-base font-bold">{formatRuntime(ups.battery_runtime)}</span>
										</div>
										<span class="text-xs text-muted-foreground mt-1">Runtime</span>
									</div>

									<!-- Load Donut Chart -->
									<div class="flex flex-col items-center">
										<div class="relative w-20 h-20">
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
											<div class="absolute inset-0 flex flex-col items-center justify-center">
												<span class="text-base font-bold">
													{ups.load}%
												</span>
											</div>
										</div>
										<span class="text-xs text-muted-foreground mt-1">Load</span>
									</div>
								</div>

								<!-- Stats Grid -->
								<div class="grid grid-cols-2 gap-3 mt-4 pt-4 border-t">
									<!-- Power Draw -->
									<div class="flex items-center gap-2">
										<Activity class="h-4 w-4 text-muted-foreground" />
										<div>
											<p class="text-xs text-muted-foreground">Power</p>
											<p class="text-sm font-medium">
												{#if ups.estimated_wattage > 0}
													{formatWattage(ups.estimated_wattage)}
												{:else if ups.power > 0}
													{formatWattage(ups.power)}
												{:else}
													--
												{/if}
												{#if ups.nominal > 0}
													<span class="text-xs text-muted-foreground">/ {formatWattage(ups.nominal)}</span>
												{/if}
											</p>
										</div>
									</div>

									<!-- Input Voltage -->
									<div class="flex items-center gap-2">
										<Zap class="h-4 w-4 text-muted-foreground" />
										<div>
											<p class="text-xs text-muted-foreground">Input</p>
											<p class="text-sm font-medium">
												{ups.input_voltage > 0 ? `${ups.input_voltage.toFixed(0)}V` : '--'}
												{#if ups.input_frequency > 0}
													<span class="text-xs text-muted-foreground">{ups.input_frequency.toFixed(0)}Hz</span>
												{/if}
											</p>
										</div>
									</div>

									<!-- Output Voltage -->
									<div class="flex items-center gap-2">
										<Gauge class="h-4 w-4 text-muted-foreground" />
										<div>
											<p class="text-xs text-muted-foreground">Output</p>
											<p class="text-sm font-medium">
												{ups.output_voltage > 0 ? `${ups.output_voltage.toFixed(0)}V` : '--'}
												{#if ups.output_frequency > 0}
													<span class="text-xs text-muted-foreground">{ups.output_frequency.toFixed(0)}Hz</span>
												{/if}
											</p>
										</div>
									</div>

									<!-- Battery Voltage -->
									{#if ups.battery_voltage > 0}
										<div class="flex items-center gap-2">
											<Battery class="h-4 w-4 text-muted-foreground" />
											<div>
												<p class="text-xs text-muted-foreground">Batt. Voltage</p>
												<p class="text-sm font-medium">{ups.battery_voltage.toFixed(1)}V</p>
											</div>
										</div>
									{/if}

									<!-- Temperature -->
									{#if ups.temperature > 0}
										<div class="flex items-center gap-2">
											<Thermometer class="h-4 w-4 text-muted-foreground" />
											<div>
												<p class="text-xs text-muted-foreground">Temperature</p>
												<p class="text-sm font-medium">{ups.temperature.toFixed(0)}°C</p>
											</div>
										</div>
									{/if}

									<!-- Current -->
									{#if ups.current > 0}
										<div class="flex items-center gap-2">
											<Activity class="h-4 w-4 text-muted-foreground" />
											<div>
												<p class="text-xs text-muted-foreground">Current</p>
												<p class="text-sm font-medium">{ups.current.toFixed(1)}A</p>
											</div>
										</div>
									{/if}
								</div>

								<!-- Raw Status -->
								<div class="mt-3 pt-3 border-t">
									<div class="flex items-center gap-2 text-xs text-muted-foreground">
										<Info class="h-3 w-3" />
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
			<Button variant="outline" size="sm" href="/devices">
				Manage Devices
			</Button>
		</div>

		{#if loading}
			<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				{#each [1, 2] as _}
					<Card.Root>
						<Card.Content class="p-6">
							<div class="animate-pulse space-y-2">
								<div class="h-4 bg-muted rounded w-1/2"></div>
								<div class="h-3 bg-muted rounded w-3/4"></div>
							</div>
						</Card.Content>
					</Card.Root>
				{/each}
			</div>
		{:else if devices.length === 0}
			<Card.Root>
				<Card.Content class="p-6 text-center text-muted-foreground">
					<Power class="h-12 w-12 mx-auto mb-4 opacity-50" />
					<p>No devices configured.</p>
					<Button variant="link" href="/devices" class="mt-2">Add Device</Button>
				</Card.Content>
			</Card.Root>
		{:else}
			<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				{#each devices as device}
					<Card.Root>
						<Card.Content class="p-4 flex items-center justify-between">
							<div>
								<h3 class="font-medium">{device.name}</h3>
								<p class="text-sm text-muted-foreground font-mono">{device.mac}</p>
							</div>
							<Button
								size="sm"
								onclick={() => wakeDevice(device)}
								disabled={wakingDevice === device.id}
							>
								{#if wakingDevice === device.id}
									<RefreshCw class="h-4 w-4 mr-1 animate-spin" />
								{:else}
									<Power class="h-4 w-4 mr-1" />
								{/if}
								Wake
							</Button>
						</Card.Content>
					</Card.Root>
				{/each}
			</div>
		{/if}
	</section>
</div>
