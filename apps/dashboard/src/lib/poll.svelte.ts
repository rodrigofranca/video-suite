/** Polling interval for status updates (docs/adr/0011). */
export const POLL_MS = 3000;

/**
 * Calls `tick` now and then every POLL_MS while `active()` is true. Call it
 * from a component: the timer stops when the component is destroyed.
 */
export function poll(tick: () => Promise<void>, active: () => boolean = () => true) {
	$effect(() => {
		let stopped = false;
		let timer: ReturnType<typeof setTimeout>;
		const run = async () => {
			await tick();
			if (!stopped && active()) timer = setTimeout(run, POLL_MS);
		};
		run();
		return () => {
			stopped = true;
			clearTimeout(timer);
		};
	});
}
