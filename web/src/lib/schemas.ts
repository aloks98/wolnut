import { z } from 'zod';

// MAC address regex: accepts AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF
const macAddressRegex = /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$/;

// IP address regex: basic IPv4 validation
const ipAddressRegex =
	/^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;

// Device schema
export const deviceSchema = z.object({
	name: z.string().min(1, 'Name is required').max(100, 'Name is too long'),
	mac: z
		.string()
		.min(1, 'MAC address is required')
		.regex(
			macAddressRegex,
			'Invalid MAC address format (use AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF)'
		),
	ip: z
		.string()
		.refine((val) => val === '' || ipAddressRegex.test(val), {
			message: 'Invalid IP address format'
		})
		.default('')
});

// UPS schema
export const upsSchema = z.object({
	name: z.string().min(1, 'Display name is required').max(100, 'Name is too long'),
	host: z
		.string()
		.min(1, 'Host is required')
		.refine(
			(val) => {
				// Accept hostname:port or just hostname
				const parts = val.split(':');
				if (parts.length > 2) return false;
				if (parts.length === 2) {
					const port = parseInt(parts[1], 10);
					if (isNaN(port) || port < 1 || port > 65535) return false;
				}
				return parts[0].length > 0;
			},
			{ message: 'Invalid host format (use hostname or hostname:port)' }
		),
	ups_name: z.string().min(1, 'UPS name is required').max(50, 'UPS name is too long')
});

export type DeviceFormData = z.infer<typeof deviceSchema>;
export type UPSFormData = z.infer<typeof upsSchema>;
