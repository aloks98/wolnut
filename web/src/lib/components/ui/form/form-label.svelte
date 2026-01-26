<script lang="ts">
	import * as FormPrimitive from "formsnap";
	import { Label } from "$lib/components/ui/label/index.js";
	import { cn } from "$lib/utils.js";
	import type { Snippet } from "svelte";

	let {
		ref = $bindable(null),
		children,
		class: className,
		...restProps
	}: { ref?: HTMLLabelElement | null; children?: Snippet; class?: string } = $props();
</script>

<FormPrimitive.Label {...restProps}>
	{#snippet child({ props })}
		<Label
			bind:ref
			{...props}
			data-slot="form-label"
			class={cn("data-[fs-error]:text-destructive", className)}
		>
			{@render children?.()}
		</Label>
	{/snippet}
</FormPrimitive.Label>
