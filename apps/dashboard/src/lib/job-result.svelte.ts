/**
 * Something a page reads once a job finishes (docs/adr/0037): the transcript
 * after the transcription, the summary after the analysis, the clips after
 * the suggestions. It loads whenever `key` gives a new value, so each new run
 * loads it again and polling alone does not. `missing` is true when the job
 * finished but there is nothing to show (the API answered 404).
 *
 * Create it while a component starts: it loads from an effect.
 */
export class JobResult<T> {
	value = $state<T | null>(null);
	missing = $state(false);
	#loadedFor = '';
	#load: () => Promise<T>;

	constructor(key: () => string | null, load: () => Promise<T>) {
		this.#load = load;
		$effect(() => {
			const k = key();
			if (!k || k === this.#loadedFor) return;
			this.#loadedFor = k;
			this.reload();
		});
	}

	/** Loads again now, after a change made on the page. */
	async reload(): Promise<void> {
		try {
			this.value = await this.#load();
			this.missing = false;
		} catch {
			this.value = null;
			this.missing = true;
		}
	}
}
