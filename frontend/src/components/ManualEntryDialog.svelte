<script>
    import { createEventDispatcher, onMount } from 'svelte';

    export let file;
    export let index = 1;
    export let total = 1;

    const dispatch = createEventDispatcher();

    $: metaArtist = file.cleanMetaArtist || '';
    $: metaTitle = file.cleanMetaTitle || '';
    $: fileArtist = file.artist || '';
    $: fileTitle = file.title || '';
    $: hasMeta = metaArtist || metaTitle;
    $: hasFile = fileArtist || fileTitle;
    $: fileSummary = (fileArtist || '?') + ' \u2014 ' + (fileTitle || '?');
    $: metaSummary = (metaArtist || '?') + ' \u2014 ' + (metaTitle || '?');

    let artist = metaArtist || fileArtist || '';
    let title = metaTitle || fileTitle || '';
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
                {#if metaArtist}
                    <button class="btn-src btn-meta" on:click={() => artist = metaArtist} title="Use from tags">&larr; tags</button>
                {/if}
                {#if fileArtist}
                    <button class="btn-src btn-file" on:click={() => artist = fileArtist} title="Use from filename">&larr; file</button>
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
                {#if metaTitle}
                    <button class="btn-src btn-meta" on:click={() => title = metaTitle} title="Use from tags">&larr; tags</button>
                {/if}
                {#if fileTitle}
                    <button class="btn-src btn-file" on:click={() => title = fileTitle} title="Use from filename">&larr; file</button>
                {/if}
            </div>
        </div>

        <div class="dialog-actions">
            {#if total > 1 && hasMeta}
                <div class="batch-actions">
                    {#if metaArtist}
                        <button class="btn-batch" on:click={() => dispatch('batch', { useArtist: true, useTitle: false })}>
                            Tags artists &rarr; all
                        </button>
                    {/if}
                    {#if metaTitle}
                        <button class="btn-batch" on:click={() => dispatch('batch', { useArtist: false, useTitle: true })}>
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
</div>

<style>
    .choice-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 100;
    }

    .choice-dialog {
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 24px;
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .choice-title {
        margin: 0 0 4px;
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        text-align: center;
    }

    .manual-dialog {
        width: 420px;
    }

    .dialog-filename {
        margin: 0;
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
        margin-bottom: 2px;
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

    .dialog-actions {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 8px;
        margin-top: 4px;
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
