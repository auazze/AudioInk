<script>
    import { onMount } from 'svelte';
    import { GetPendingFiles, ConfirmSubmit, ConfirmSkip, ConfirmDone, ConfirmBatchFromTags } from '../../wailsjs/go/main/App.js';

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

    async function batchFromTags(useArtist, useTitle) {
        await ConfirmBatchFromTags(currentIndex, useArtist, useTitle);
        await ConfirmDone();
    }

    function handleKeydown(e) {
        if (e.key === 'Escape') handleSkip();
        if (e.key === 'Enter' && (artist.trim() || title.trim())) handleSubmit();
    }

    $: current = pendingFiles[currentIndex];
    $: hasMeta = current && (current.metaArtist || current.metaTitle);
    $: hasFile = current && (current.fileArtist || current.fileTitle);
    $: fileSummary = current
        ? (current.fileArtist || '?') + ' \u2014 ' + (current.fileTitle || '?')
        : '';
    $: metaSummary = current
        ? (current.metaArtist || '?') + ' \u2014 ' + (current.metaTitle || '?')
        : '';
    $: hasAnyMetaArtist = pendingFiles.some((f, i) => i >= currentIndex && f.metaArtist);
    $: hasAnyMetaTitle = pendingFiles.some((f, i) => i >= currentIndex && f.metaTitle);
    $: remaining = pendingFiles.length - currentIndex;
</script>

<div class="confirm-root" on:keydown={handleKeydown}>
    {#if current}
        <div class="confirm-card">
            <p class="confirm-header">Manual entry ({currentIndex + 1}/{pendingFiles.length})</p>
            <p class="confirm-filename" title={current.filename}>{current.filename}</p>

            {#if hasFile || hasMeta}
                <div class="sources">
                    {#if hasFile}
                        <div class="source-row source-file">
                            <span class="source-label">File:</span>
                            <span class="source-value" title={fileSummary}>{fileSummary}</span>
                        </div>
                    {/if}
                    {#if hasMeta}
                        <div class="source-row source-meta">
                            <span class="source-label">Tags:</span>
                            <span class="source-value" title={metaSummary}>{metaSummary}</span>
                        </div>
                    {/if}
                </div>
            {/if}

            <div class="field-group">
                <span class="field-label-text">Artist</span>
                <div class="field-row">
                    <input
                        class="field-input"
                        bind:this={artistInput}
                        bind:value={artist}
                        placeholder="Artist name"
                    />
                    {#if current.metaArtist}
                        <button class="btn-src btn-meta" on:click={() => artist = current.metaArtist} title="Use from tags">&larr; tags</button>
                    {/if}
                    {#if current.fileArtist}
                        <button class="btn-src btn-file" on:click={() => artist = current.fileArtist} title="Use from filename">&larr; file</button>
                    {/if}
                </div>
            </div>

            <div class="field-group">
                <span class="field-label-text">Title</span>
                <div class="field-row">
                    <input
                        class="field-input"
                        bind:value={title}
                        placeholder="Track title"
                    />
                    {#if current.metaTitle}
                        <button class="btn-src btn-meta" on:click={() => title = current.metaTitle} title="Use from tags">&larr; tags</button>
                    {/if}
                    {#if current.fileTitle}
                        <button class="btn-src btn-file" on:click={() => title = current.fileTitle} title="Use from filename">&larr; file</button>
                    {/if}
                </div>
            </div>

            <div class="confirm-actions">
                {#if remaining > 1 && (hasAnyMetaArtist || hasAnyMetaTitle)}
                    <div class="batch-actions">
                        {#if hasAnyMetaArtist}
                            <button class="btn-batch" on:click={() => batchFromTags(true, false)}>
                                Tags artists &rarr; all
                            </button>
                        {/if}
                        {#if hasAnyMetaTitle}
                            <button class="btn-batch" on:click={() => batchFromTags(false, true)}>
                                Tags titles &rarr; all
                            </button>
                        {/if}
                    </div>
                {/if}
                <div class="main-actions">
                    <button class="btn-skip" on:click={handleSkip}>Skip</button>
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
        width: 420px;
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

    .sources {
        display: flex;
        flex-direction: column;
        gap: 3px;
        margin-bottom: 10px;
    }

    .source-row {
        padding: 5px 10px;
        border-radius: 5px;
        display: flex;
        gap: 6px;
        align-items: center;
        overflow: hidden;
        font-size: 11px;
    }

    .source-file {
        background: rgba(120, 180, 120, 0.08);
        border: 1px solid rgba(120, 180, 120, 0.2);
    }

    .source-meta {
        background: rgba(99, 102, 241, 0.08);
        border: 1px solid rgba(99, 102, 241, 0.2);
    }

    .source-label {
        font-weight: 600;
        flex-shrink: 0;
    }

    .source-file .source-label { color: var(--green, #78b478); }
    .source-meta .source-label { color: var(--accent); }

    .source-value {
        color: var(--text);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .field-group {
        margin-bottom: 10px;
    }

    .field-label-text {
        display: block;
        font-size: 12px;
        color: var(--text-dim);
        margin-bottom: 4px;
    }

    .field-row {
        display: flex;
        gap: 4px;
        align-items: center;
    }

    .field-input {
        flex: 1;
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

    .btn-src {
        padding: 5px 7px;
        border-radius: 5px;
        font-size: 10px;
        cursor: pointer;
        transition: all 0.15s;
        white-space: nowrap;
        flex-shrink: 0;
    }

    .btn-meta {
        border: 1px solid rgba(99, 102, 241, 0.3);
        background: rgba(99, 102, 241, 0.08);
        color: var(--accent);
    }
    .btn-meta:hover {
        background: rgba(99, 102, 241, 0.15);
        border-color: var(--accent);
    }

    .btn-file {
        border: 1px solid rgba(120, 180, 120, 0.3);
        background: rgba(120, 180, 120, 0.08);
        color: var(--green, #78b478);
    }
    .btn-file:hover {
        background: rgba(120, 180, 120, 0.15);
        border-color: var(--green, #78b478);
    }

    .confirm-actions {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-top: 6px;
        gap: 8px;
    }

    .batch-actions {
        display: flex;
        gap: 4px;
    }

    .btn-batch {
        padding: 5px 8px;
        border: 1px solid rgba(99, 102, 241, 0.3);
        border-radius: 6px;
        background: rgba(99, 102, 241, 0.08);
        color: var(--accent);
        font-size: 10px;
        cursor: pointer;
        transition: all 0.15s;
        white-space: nowrap;
    }

    .btn-batch:hover {
        background: rgba(99, 102, 241, 0.15);
        border-color: var(--accent);
    }

    .main-actions {
        display: flex;
        gap: 8px;
        margin-left: auto;
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
