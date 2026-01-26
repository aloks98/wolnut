import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { intervalToDuration } from 'date-fns';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

// Date/Time formatting
export function formatRuntime(seconds: number): string {
	if (seconds <= 0) return '--';
	if (seconds < 60) return `${seconds}s`;

	const duration = intervalToDuration({ start: 0, end: seconds * 1000 });
	const parts: string[] = [];

	if (duration.hours && duration.hours > 0) {
		parts.push(`${duration.hours}h`);
	}
	if (duration.minutes && duration.minutes > 0) {
		parts.push(`${duration.minutes}m`);
	}

	return parts.length > 0 ? parts.join(' ') : '0m';
}

export function formatTime(date: Date): string {
	return date.toLocaleTimeString();
}

// Number formatting
export function formatWattage(watts: number): string {
	if (watts >= 1000) {
		return `${(watts / 1000).toFixed(1)}kW`;
	}
	return `${Math.round(watts)}W`;
}

// Color utilities for status indicators
export function getBatteryColor(charge: number): string {
	if (charge > 50) return '#22c55e'; // green-500
	if (charge > 20) return '#f59e0b'; // amber-500
	return '#ef4444'; // red-500
}

export function getBatteryTextColor(charge: number): string {
	if (charge > 50) return 'text-emerald-500';
	if (charge > 20) return 'text-amber-500';
	return 'text-red-500';
}

export function getLoadColor(load: number): string {
	if (load < 50) return '#22c55e'; // green
	if (load < 80) return '#f59e0b'; // amber
	return '#ef4444'; // red
}

export function getDeviceStatusColor(online: boolean | null): string {
	if (online === null) return 'text-muted-foreground';
	return online ? 'text-emerald-500' : 'text-red-500';
}

export function getDeviceStatusTitle(online: boolean | null): string {
	if (online === null) return 'No IP configured';
	return online ? 'Online' : 'Offline';
}

// Type helpers for shadcn-svelte components
export type WithElementRef<T, E extends HTMLElement = HTMLElement> = T & {
	ref?: E | null;
};

export type WithoutChildrenOrChild<T> = Omit<T, "children" | "child">;

export type WithoutChildren<T> = Omit<T, "children">;

export type WithoutChild<T> = Omit<T, "child">;
