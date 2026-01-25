// Device types
export interface Device {
	id: string;
	name: string;
	mac: string;
}

// UPS types
export interface UPSEntry {
	id: string;
	name: string;
	host: string;
	ups_name: string;
}

export interface UPSStatus {
	id: string;
	name: string;
	host: string;

	// Connection status
	online: boolean;
	error?: string;

	// UPS status
	status: string;
	status_label: string;
	is_online: boolean;
	is_on_battery: boolean;
	is_low_battery: boolean;
	is_charging: boolean;

	// Battery
	battery_charge: number;
	battery_voltage: number;
	battery_runtime: number;

	// Load
	load: number;
	power: number;
	nominal: number;
	current: number;

	// Input
	input_voltage: number;
	input_frequency: number;

	// Output
	output_voltage: number;
	output_frequency: number;

	// UPS Info
	model: string;
	manufacturer: string;
	serial: string;
	firmware: string;

	// Temperature
	temperature: number;

	// Calculated
	estimated_wattage: number;
}

// API response types
export interface APIResponse<T> {
	success: boolean;
	data?: T;
	error?: string;
}

// Request types
export interface CreateDeviceRequest {
	name: string;
	mac: string;
}

export interface CreateUPSRequest {
	name: string;
	host: string;
	ups_name: string;
}
