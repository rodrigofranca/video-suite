/** Polling interval for status updates (docs/adr/0011). */
export const POLL_MS = 3000;

/**
 * Calls `tick` now and then every POLL_MS while `active()` is true. Call it
 * from a component: the timer stops when the component is destroyed. The
 * returned `refresh` runs `tick` at once and resumes polling if it stopped.
 */
export function poll(tick: () => Promise<void>, active: () => boolean = () => true) {
	let stopped = false;
	let timer: ReturnType<typeof setTimeout> | undefined;
	let latest = 0; // only the most recent run schedules the next one

	const run = async () => {
		const mine = ++latest;
		clearTimeout(timer);
		await tick();
		if (!stopped && mine === latest && active()) timer = setTimeout(run, POLL_MS);
	};

	$effect(() => {
		stopped = false;
		run();
		return () => {
			stopped = true;
			clearTimeout(timer);
		};
	});

	return { refresh: run };
}
