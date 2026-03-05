<script>
    import { onMount } from 'svelte';
    import { GetPendingFiles, ConfirmSubmit, ConfirmSkip, ConfirmDone } from '../../wailsjs/go/main/App.js';

    let pendingFiles = [];
    let currentIndex = 0;
    let artist = '';
    let title = '';
    let artistInput;

    onMount(async () => {
        pendingFiles = await GetPendingFiles();
        if (pendingFiles && pendingFiles.length > 0) {
            loadCurrent();
        } else {
            await ConfirmDone();
        }
    });

    function loadCurrent() {
        if (currentIndex < pendingFiles.length) {
            artist = pendingFiles[currentIndex].artist || '';
            title = pendingFiles[currentIndex].title || '';
            if (artistInput) artistInput.focus();
        }
    }

    async function handleSubmit() {
        if (!artist.trim() && !title.trim()) return;
        await ConfirmSubmit(currentIndex, artist.trim(), title.trim());
        advance();
    }

    async function handleSkip() {
        await ConfirmSkip(currentIndex);
        advance();
    }

    async function advance() {
        currentIndex++;
        if (currentIndex >= pendingFiles.length) {
            await ConfirmDone();
        } else {
            loadCurrent();
        }
    }

    function handleKeydown(e) {
        if (e.key === 'Escape') handleSkip();
        if (e.key === 'Enter' && (artist.trim() || title.trim())) handleSubmit();
    }

    $: current = pendingFiles[currentIndex];
    $: suggestion = current
        ? (current.artist && current.title
            ? current.artist + ' - ' + current.title
            : current.title || '')
        : '';
</script>

<div class="confirm-root" on:keydown={handleKeydown}>
    {#if current}
        <div class="confirm-card">
            <p class="confirm-header">Manual entry ({currentIndex + 1}/{pendingFiles.length})</p>
            <p class="confirm-filename" title={current.filename}>{current.filename}</p>

            {#if suggestion}
                <p class="confirm-suggestion">{suggestion}</p>
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

            <div class="confirm-actions">
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
    {/if}
</div>

<style>
    .confirm-root {
        height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg);
    }

    .confirm-card {
        width: 380px;
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 24px;
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .confirm-header {
        margin: 0 0 4px;
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        text-align: center;
    }

    .confirm-filename {
        margin: 0 0 8px;
        font-size: 12px;
        color: var(--text-dim);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        text-align: center;
    }

    .confirm-suggestion {
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

    .confirm-actions {
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
