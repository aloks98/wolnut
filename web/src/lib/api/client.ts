import type {
	Device,
	DeviceStatus,
	UPSEntry,
	UPSStatus,
	APIResponse,
	CreateDeviceRequest,
	CreateUPSRequest
} from './types';

const BASE_URL = '/api';

async function fetchAPI<T>(
	endpoint: string,
	options?: RequestInit
): Promise<APIResponse<T>> {
	const isFormData = options?.body instanceof FormData;
	// FormData sets its own Content-Type with the multipart boundary —
	// don't override it.
	const baseHeaders: HeadersInit = isFormData
		? { ...options?.headers }
		: { 'Content-Type': 'application/json', ...options?.headers };

	try {
		const response = await fetch(`${BASE_URL}${endpoint}`, {
			...options,
			headers: baseHeaders
		});

		if (!response.ok) {
			let detail = '';
			try {
				const body = await response.text();
				const parsed = body ? JSON.parse(body) : null;
				detail = parsed?.error || body.slice(0, 200);
			} catch {
				// non-JSON body (e.g. proxy 502 HTML) — leave detail empty
			}
			return {
				success: false,
				error: detail
					? `HTTP ${response.status}: ${detail}`
					: `HTTP ${response.status}`
			};
		}

		return await response.json();
	} catch (error) {
		return {
			success: false,
			error: error instanceof Error ? error.message : 'Unknown error'
		};
	}
}

// Device API
export const deviceAPI = {
	getAll: () => fetchAPI<Device[]>('/devices'),

	getStatus: () => fetchAPI<DeviceStatus[]>('/devices/status'),

	create: (device: CreateDeviceRequest) =>
		fetchAPI<Device>('/devices', {
			method: 'POST',
			body: JSON.stringify(device)
		}),

	update: (id: string, device: CreateDeviceRequest) =>
		fetchAPI<Device>(`/devices/${id}`, {
			method: 'PUT',
			body: JSON.stringify(device)
		}),

	delete: (id: string) =>
		fetchAPI<null>(`/devices/${id}`, {
			method: 'DELETE'
		}),

	wake: (id: string) =>
		fetchAPI<{ message: string }>(`/devices/${id}/wake`, {
			method: 'POST'
		})
};

// UPS API
export const upsAPI = {
	getAll: () => fetchAPI<UPSEntry[]>('/ups'),

	create: (ups: CreateUPSRequest) =>
		fetchAPI<UPSEntry>('/ups', {
			method: 'POST',
			body: JSON.stringify(ups)
		}),

	update: (id: string, ups: CreateUPSRequest) =>
		fetchAPI<UPSEntry>(`/ups/${id}`, {
			method: 'PUT',
			body: JSON.stringify(ups)
		}),

	delete: (id: string) =>
		fetchAPI<null>(`/ups/${id}`, {
			method: 'DELETE'
		}),

	getStatus: () => fetchAPI<UPSStatus[]>('/ups/status')
};

// Config API
export const configAPI = {
	exportUrl: `${BASE_URL}/config/export`,

	import: (file: File) => {
		const formData = new FormData();
		formData.append('file', file);
		return fetchAPI<{ message: string }>('/config/import', {
			method: 'POST',
			body: formData
		});
	}
};

// Health API
export const healthAPI = {
	check: () => fetchAPI<{ status: string }>('/health')
};

// Version API
export const versionAPI = {
	getCurrent: () => fetchAPI<{ version: string; commit: string }>('/version'),

	// Proxied through the Go backend (with TTL cache) so we don't burn the
	// unauthenticated GitHub rate limit per browser tab.
	getLatest: async (): Promise<{ version: string; url: string } | null> => {
		const res = await fetchAPI<{ version: string; url: string }>('/version/latest');
		if (res.success && res.data) return res.data;
		return null;
	}
};
