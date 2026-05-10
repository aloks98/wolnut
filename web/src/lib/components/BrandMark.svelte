<script lang="ts">
	// Solid dot with two concentric ripples — represents the wake signal pulse
	// being broadcast.  Inherits text color via currentColor, so any wrapper
	// class (e.g. text-brand) tints the whole mark.
	let { class: cls = '' }: { class?: string } = $props();
</script>

<svg viewBox="0 0 24 24" class={cls} fill="none" aria-hidden="true">
	<circle cx="12" cy="12" r="2.25" fill="currentColor" />
	<circle
		cx="12"
		cy="12"
		r="5.5"
		stroke="currentColor"
		stroke-width="1.5"
		opacity="0.55"
	/>
	<circle
		cx="12"
		cy="12"
		r="9"
		stroke="currentColor"
		stroke-width="1.5"
		opacity="0.25"
	/>
</svg>
