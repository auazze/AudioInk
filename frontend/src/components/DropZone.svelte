<script>
    import { createEventDispatcher } from 'svelte';

    const dispatch = createEventDispatcher();
</script>

<!-- Wails uses --wails-drop-target: drop to identify this as a drop target -->
<!-- Wails adds class "wails-drop-target-active" during drag-over -->
<div class="dropzone" style="--wails-drop-target: drop">
    <div class="dropzone-content">
        <div class="dropzone-icon">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
        </div>
        <div class="dropzone-text">Drag & drop audio files here</div>
        <div class="dropzone-buttons">
            <button class="btn-select" on:click={() => dispatch('selectFiles')}>
                Select files
            </button>
            <button class="btn-folder" on:click={() => dispatch('selectFolder')}>
                Select folder
            </button>
        </div>
        <div class="dropzone-hint">MP3, FLAC, OGG, M4A, WAV, WMA, OPUS</div>
    </div>
</div>

<style>
    .dropzone {
        flex: 1;
        margin: 0 16px 16px;
        border: 2px dashed var(--border);
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s ease;
        min-height: 0;
    }

    /* Wails adds this class when dragging files over the drop target */
    :global(.wails-drop-target-active).dropzone {
        border-color: var(--accent);
        border-style: solid;
        background: rgba(124, 106, 239, 0.08);
    }

    .dropzone-content {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 12px;
    }

    .dropzone-icon {
        color: var(--text-dim);
        transition: color 0.2s;
    }

    :global(.wails-drop-target-active) .dropzone-icon {
        color: var(--accent);
    }

    .dropzone-text {
        font-size: 16px;
        font-weight: 500;
        color: var(--text);
    }

    .dropzone-buttons {
        display: flex;
        gap: 10px;
        margin-top: 4px;
    }

    .btn-select, .btn-folder {
        padding: 8px 20px;
        border-radius: 6px;
        font-size: 13px;
        cursor: pointer;
        transition: all 0.15s;
    }

    .btn-select {
        background: var(--accent);
        color: white;
        border: none;
        font-weight: 600;
    }

    .btn-select:hover {
        background: var(--accent-hover);
    }

    .btn-folder {
        background: transparent;
        color: var(--text-dim);
        border: 1px solid var(--border);
    }

    .btn-folder:hover {
        border-color: var(--text-dim);
        color: var(--text);
    }

    .dropzone-hint {
        font-size: 11px;
        color: var(--text-dim);
        letter-spacing: 0.03em;
        margin-top: 4px;
    }
</style>
