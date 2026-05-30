import { z } from 'zod';

// MAC address regex: accepts AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF
const macAddressRegex = /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$/;

// Accept IPv4 or IPv6 — the backend validates with net.ParseIP (both families)
// and the online-status probe handles either via net.JoinHostPort, so the form
// must not reject a valid IPv6 address.
function isValidIP(val: string): boolean {
	return z.ipv4().safeParse(val).success || z.ipv6().safeParse(val).success;
}

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
		.refine((val) => val === '' || isValidIP(val), {
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
