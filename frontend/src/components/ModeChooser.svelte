<script>
    import { onMount } from 'svelte';
    import { GetChooserFileCount, ChooseGUI, ChooseAutoFix } from '../../wailsjs/go/main/App.js';

    let fileCount = 0;

    onMount(async () => {
        fileCount = await GetChooserFileCount();
    });
</script>

<div class="chooser-root">
    <div class="chooser-card">
        <span class="chooser-title">AudioInk</span>
        <span class="chooser-count">{fileCount} audio {fileCount === 1 ? 'file' : 'files'}</span>

        <button class="chooser-btn" on:click={ChooseGUI}>
            <span class="btn-label">Open in AudioInk</span>
            <span class="btn-desc">Fine-tune each file individually</span>
        </button>

        <button class="chooser-btn chooser-btn-accent" on:click={ChooseAutoFix}>
            <span class="btn-label">Auto-fix</span>
            <span class="btn-desc">Metadata &rarr; filename, ask only if needed</span>
        </button>
    </div>
</div>

<style>
    .chooser-root {
        height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg);
    }

    .chooser-card {
        width: 290px;
        display: flex;
        flex-direction: column;
        gap: 10px;
        text-align: center;
    }

    .chooser-title {
        font-size: 16px;
        font-weight: 700;
        letter-spacing: -0.02em;
        background: linear-gradient(135deg, var(--accent), #a78bfa);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .chooser-count {
        font-size: 12px;
        color: var(--text-dim);
        margin-bottom: 4px;
    }

    .chooser-btn {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 2px;
        padding: 14px 16px;
        border: 1px solid var(--border);
        border-radius: 8px;
        background: transparent;
        cursor: pointer;
        transition: all 0.15s;
        text-align: left;
    }

    .chooser-btn:hover {
        border-color: var(--accent);
        background: rgba(99, 102, 241, 0.05);
    }

    .chooser-btn-accent {
        border-color: rgba(99, 102, 241, 0.3);
        background: rgba(99, 102, 241, 0.05);
    }

    .chooser-btn-accent:hover {
        background: rgba(99, 102, 241, 0.1);
    }

    .btn-label {
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
    }

    .btn-desc {
        font-size: 11px;
        color: var(--text-dim);
    }
</style>
