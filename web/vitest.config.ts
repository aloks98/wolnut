import { defineConfig } from 'vitest/config';

// Unit tests for pure helpers in src/lib. No SvelteKit plugin: the functions
// under test are plain TS, so a lightweight node environment keeps runs fast.
export default defineConfig({
	test: {
		environment: 'node',
		include: ['src/**/*.{test,spec}.ts']
	}
});
