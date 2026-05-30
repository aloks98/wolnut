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

	// Roll days/months/years up into hours — a UPS runtime is always shown in
	// h/m, and intervalToDuration buckets anything ≥24h into `days` (and beyond),
	// which would otherwise be silently dropped (25h → "1h").
	const totalHours =
		(duration.years ?? 0) * 365 * 24 +
		(duration.months ?? 0) * 30 * 24 +
		(duration.days ?? 0) * 24 +
		(duration.hours ?? 0);

	if (totalHours > 0) {
		parts.push(`${totalHours}h`);
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

// Status helpers — semantic palette per app.css:
//   healthy → emerald-500 · warning → amber-500 · critical → red-500 · unknown → muted

// Standard badge recipe for online/offline state.  Used on the dashboard
// device row and the devices page status column so both render identically.
export function deviceStatusBadgeClass(online: boolean | null | undefined): string {
	if (online === null || online === undefined) {
		return 'bg-muted/50 text-muted-foreground border-muted';
	}
	if (online) return 'bg-emerald-500/15 text-emerald-500 border-emerald-500/20';
	return 'bg-red-500/15 text-red-500 border-red-500/20';
}

export function deviceStatusLabel(online: boolean | null | undefined): string {
	if (online === null || online === undefined) return 'Unknown';
	return online ? 'Online' : 'Offline';
}

// Type helpers for shadcn-svelte components
export type WithElementRef<T, E extends HTMLElement = HTMLElement> = T & {
	ref?: E | null;
};

export type WithoutChildrenOrChild<T> = Omit<T, "children" | "child">;

export type WithoutChildren<T> = Omit<T, "children">;

export type WithoutChild<T> = Omit<T, "child">;
