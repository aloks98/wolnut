import { describe, it, expect } from 'vitest';
import { formatRuntime, formatWattage, compareVersions } from './utils';

describe('formatRuntime', () => {
	it('renders sub-minute and placeholder values', () => {
		expect(formatRuntime(0)).toBe('--');
		expect(formatRuntime(-10)).toBe('--');
		expect(formatRuntime(45)).toBe('45s');
	});

	it('renders minutes and hours', () => {
		expect(formatRuntime(60)).toBe('1m');
		expect(formatRuntime(90)).toBe('1m'); // seconds dropped past 1 minute
		expect(formatRuntime(3600)).toBe('1h');
		expect(formatRuntime(3660)).toBe('1h 1m');
	});

	// Regression: intervalToDuration buckets ≥24h into `days`, which used to be
	// dropped — a 25h runtime rendered as "1h".
	it('rolls days into the hours count (no dropped time)', () => {
		expect(formatRuntime(25 * 3600)).toBe('25h');
		expect(formatRuntime(25 * 3600 + 30 * 60)).toBe('25h 30m');
		expect(formatRuntime(48 * 3600)).toBe('48h');
	});
});

describe('formatWattage', () => {
	it('formats watts and kilowatts', () => {
		expect(formatWattage(0)).toBe('0W');
		expect(formatWattage(250.4)).toBe('250W');
		expect(formatWattage(1500)).toBe('1.5kW');
	});
});

describe('compareVersions', () => {
	it('orders by numeric segment', () => {
		expect(compareVersions('1.2.0', '1.1.9')).toBe(1);
		expect(compareVersions('1.1.0', '1.2.0')).toBe(-1);
		expect(compareVersions('1.2.3', '1.2.3')).toBe(0);
		expect(compareVersions('1.10.0', '1.9.0')).toBe(1); // not string-compared
	});

	it('ignores a leading v and differing segment counts', () => {
		expect(compareVersions('v2.0.0', '1.9.9')).toBe(1);
		expect(compareVersions('1.2', '1.2.0')).toBe(0);
		expect(compareVersions('1.2.1', '1.2')).toBe(1);
	});

	// Regression: '3-beta' → NaN previously poisoned the comparison so any
	// suffixed tag reported "no update".
	it('treats a pre-release suffix as its release base', () => {
		expect(compareVersions('1.2.3-beta', '1.2.3')).toBe(0);
		expect(compareVersions('1.2.4-rc1', '1.2.3')).toBe(1);
	});
});
