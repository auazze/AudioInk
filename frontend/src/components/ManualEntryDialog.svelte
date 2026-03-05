<script>
    import { createEventDispatcher, onMount } from 'svelte';

    export let file;
    export let index = 1;
    export let total = 1;

    const dispatch = createEventDispatcher();

    let artist = file.artist || '';
    let title = file.title || '';
    let artistInput;


    onMount(() => {
        if (artistInput) artistInput.focus();
    });

    function handleSubmit() {
        if (!artist.trim() && !title.trim()) return;
        dispatch('submit', { artist: artist.trim(), title: title.trim() });
    }

    function handleSkip() {
        dispatch('skip');
    }

    function handleKeydown(e) {
        if (e.key === 'Escape') handleSkip();
        if (e.key === 'Enter' && (artist.trim() || title.trim())) handleSubmit();
    }
</script>

<div class="choice-overlay" on:keydown={handleKeydown}>
    <div class="choice-dialog manual-dialog">
        <p class="choice-title">Manual entry ({index}/{total})</p>
        <p class="dialog-filename" title={file.filename}>{file.filename}</p>

        {#if file.artist || file.title}
            <p class="dialog-suggestion">
                {file.artist || '?'} — {file.title || '?'}
            </p>
        {/if}

        <label class="field-label">
            Artist
            <input
                class="field-input"
                bind:this={artistInput}
                bind:value={artist}
                placeholder="Artist name"
            />
        </label>

        <label class="field-label">
            Title
            <input
                class="field-input"
                bind:value={title}
                placeholder="Track title"
            />
        </label>

        <div class="dialog-actions">
            <button class="btn-skip" on:click={handleSkip}>
                Skip
            </button>
            <button
                class="btn-submit"
                on:click={handleSubmit}
                disabled={!artist.trim() && !title.trim()}
            >
                Submit
            </button>
        </div>
    </div>
</div>

<style>
    .manual-dialog {
        width: 380px;
    }

    .dialog-suggestion {
        margin: 0 0 12px;
        padding: 6px 10px;
        background: rgba(120, 180, 120, 0.1);
        border: 1px solid rgba(120, 180, 120, 0.25);
        border-radius: 6px;
        font-size: 12px;
        color: var(--green, #78b478);
        text-align: center;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .dialog-filename {
        margin: 0 0 8px;
        font-size: 12px;
        color: var(--text-dim);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        text-align: center;
    }

    .field-label {
        display: flex;
        flex-direction: column;
        gap: 4px;
        font-size: 12px;
        color: var(--text-dim);
        margin-bottom: 12px;
    }

    .field-input {
        width: 100%;
        padding: 8px 12px;
        background: var(--bg);
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text);
        font-size: 13px;
        outline: none;
        transition: border-color 0.15s;
        box-sizing: border-box;
    }

    .field-input:focus {
        border-color: var(--accent);
    }

    .dialog-actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        margin-top: 4px;
    }

    .btn-skip {
        padding: 6px 16px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: transparent;
        color: var(--text-dim);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.15s;
    }

    .btn-skip:hover {
        border-color: var(--text-dim);
        color: var(--text);
    }

    .btn-submit {
        padding: 6px 20px;
        border: none;
        border-radius: 6px;
        background: var(--accent);
        color: white;
        font-size: 12px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.15s;
    }

    .btn-submit:hover:not(:disabled) {
        background: var(--accent-hover);
    }

    .btn-submit:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
