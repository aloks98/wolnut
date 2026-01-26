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
	try {
		const response = await fetch(`${BASE_URL}${endpoint}`, {
			headers: {
				'Content-Type': 'application/json',
				...options?.headers
			},
			...options
		});

		const data = await response.json();
		return data;
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

	import: async (file: File) => {
		const formData = new FormData();
		formData.append('file', file);

		try {
			const response = await fetch(`${BASE_URL}/config/import`, {
				method: 'POST',
				body: formData
			});
			return await response.json();
		} catch (error) {
			return {
				success: false,
				error: error instanceof Error ? error.message : 'Unknown error'
			};
		}
	}
};

// Health API
export const healthAPI = {
	check: () => fetchAPI<{ status: string }>('/health')
};

// Version API
export const versionAPI = {
	getCurrent: () => fetchAPI<{ version: string; commit: string }>('/version'),

	getLatest: async (): Promise<{ version: string; url: string } | null> => {
		try {
			const response = await fetch(
				'https://api.github.com/repos/aloks98/wolnut/releases/latest'
			);
			if (!response.ok) return null;
			const data = await response.json();
			return {
				version: data.tag_name?.replace(/^v/, '') || data.name,
				url: data.html_url
			};
		} catch {
			return null;
		}
	}
};
