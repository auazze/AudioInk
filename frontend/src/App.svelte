<script>
    import DropZone from './components/DropZone.svelte';
    import FileTable from './components/FileTable.svelte';
    import { SelectFiles, SelectDirectory, ScanFiles, ApplyTagsCopy, ApplyTagsOverwrite, OpenOutputFolder } from '../wailsjs/go/main/App.js';
    import { OnFileDrop } from '../wailsjs/runtime/runtime.js';
    import { onMount } from 'svelte';

    let files = [];
    let applying = false;
    let appliedCount = 0;
    let errorCount = 0;
    let applyResults = [];
    let done = false;
    let showChoice = false;
    let applyMode = '';

    $: readyCount = files.filter(f => f.confidence === 'high' || f.confidence === 'medium').length;
    $: reviewCount = files.filter(f => f.confidence === 'low').length;

    onMount(() => {
        OnFileDrop((x, y, paths) => {
            handleDroppedPaths(paths);
        }, true);
    });

    async function handleDroppedPaths(paths) {
        try {
            const results = await ScanFiles(paths);
            if (results && results.length > 0) {
                files = results;
                resetState();
            }
        } catch (err) {
            console.error('scan error:', err);
        }
    }

    async function handleSelectFiles() {
        try {
            const results = await SelectFiles();
            if (results && results.length > 0) {
                files = results;
                resetState();
            }
        } catch (err) {
            console.error('file select error:', err);
        }
    }

    async function handleSelectFolder() {
        try {
            const results = await SelectDirectory();
            if (results && results.length > 0) {
                files = results;
                resetState();
            }
        } catch (err) {
            console.error('folder select error:', err);
        }
    }

    function resetState() {
        appliedCount = 0;
        errorCount = 0;
        applyResults = [];
        done = false;
        showChoice = false;
        applyMode = '';
    }

    function handleUpdate(e) {
        const { index, field, value } = e.detail;
        files[index] = { ...files[index], [field]: value, confidence: 'high' };

        const f = files[index];
        const ext = f.filename.substring(f.filename.lastIndexOf('.'));
        let name = '';
        if (f.artist && f.title) {
            name = f.artist + ' - ' + f.title;
        } else if (f.title) {
            name = f.title;
        }
        if (name && f.extras) {
            name += ' (' + f.extras + ')';
        }
        if (name) {
            files[index].newFilename = name + ext;
        }
        files = files;
    }

    function promptApply() {
        showChoice = true;
    }

    async function applyWithMode(mode) {
        showChoice = false;
        applyMode = mode;
        applying = true;
        appliedCount = 0;
        errorCount = 0;
        applyResults = [];

        const requests = files.map(f => ({
            filePath: f.filePath,
            artist: f.artist,
            title: f.title,
            extras: f.extras || '',
            track: f.track || 0,
        }));

        try {
            const applyFn = mode === 'overwrite' ? ApplyTagsOverwrite : ApplyTagsCopy;
            const results = await applyFn(requests);
            applyResults = results || [];
            for (const r of applyResults) {
                if (r.success) appliedCount++;
                else errorCount++;
            }

            for (let i = 0; i < files.length; i++) {
                const match = applyResults.find(r => r.filePath === files[i].filePath);
                if (match) {
                    files[i] = {
                        ...files[i],
                        status: match.success ? 'done' : 'error',
                        statusError: match.error || '',
                        outputFilename: match.newFilename || '',
                    };
                }
            }
            files = files;
            done = true;
        } catch (err) {
            console.error('apply error:', err);
            errorCount = files.length;
        }

        applying = false;
    }

    function openOutput() {
        if (files.length > 0) {
            OpenOutputFolder(files[0].filePath);
        }
    }

    function clearAll() {
        files = [];
        resetState();
    }
</script>

<div class="app" style="--wails-drop-target: drop">
    <header class="titlebar">
        <span class="title">AudioInk</span>
    </header>

    {#if files.length === 0}
        <DropZone on:selectFiles={handleSelectFiles} on:selectFolder={handleSelectFolder} />
    {:else}
        <FileTable {files} showStatus={done} on:update={handleUpdate} />

        {#if showChoice}
            <div class="choice-overlay" on:click|self={() => showChoice = false}>
                <div class="choice-dialog">
                    <p class="choice-title">How should files be saved?</p>
                    <button class="choice-btn choice-copy" on:click={() => applyWithMode('copy')}>
                        <span class="choice-icon">&#128230;</span>
                        <span class="choice-label">Save copies</span>
                        <span class="choice-desc">Originals stay untouched, copies go to AudioInk folder</span>
                    </button>
                    <button class="choice-btn choice-overwrite" on:click={() => applyWithMode('overwrite')}>
                        <span class="choice-icon">&#9998;</span>
                        <span class="choice-label">Fix originals</span>
                        <span class="choice-desc">Rename and tag original files in place</span>
                    </button>
                </div>
            </div>
        {/if}

        <footer class="statusbar">
            <div class="stats">
                <span>Found: <strong>{files.length}</strong></span>
                {#if !done}
                    <span class="sep">|</span>
                    <span class="stat-ready">Ready: <strong>{readyCount}</strong></span>
                    <span class="sep">|</span>
                    <span class="stat-review">Review: <strong>{reviewCount}</strong></span>
                {:else}
                    <span class="sep">|</span>
                    <span class="stat-done">Done: <strong>{appliedCount}</strong></span>
                    {#if errorCount > 0}
                        <span class="sep">|</span>
                        <span class="stat-error">Errors: <strong>{errorCount}</strong></span>
                    {/if}
                {/if}
            </div>
            <div class="actions">
                {#if done && applyMode === 'copy'}
                    <button class="btn-open" on:click={openOutput}>
                        Open output folder
                    </button>
                {/if}
                <button class="btn-reset" on:click={clearAll}>
                    Clear
                </button>
                {#if !done}
                    <button
                        class="btn-apply"
                        on:click={promptApply}
                        disabled={applying || files.length === 0}
                    >
                        {#if applying}
                            Processing...
                        {:else}
                            Apply Tags
                        {/if}
                    </button>
                {/if}
            </div>
        </footer>
    {/if}
</div>

<style>
    .app {
        height: 100vh;
        display: flex;
        flex-direction: column;
        background: var(--bg);
    }

    .titlebar {
        padding: 12px 16px 8px;
        display: flex;
        align-items: center;
        --wails-draggable: drag;
    }

    .title {
        font-size: 16px;
        font-weight: 700;
        letter-spacing: -0.02em;
        background: linear-gradient(135deg, var(--accent), #a78bfa);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .statusbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 10px 16px;
        border-top: 1px solid var(--border);
        background: var(--surface);
    }

    .stats {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 12px;
        color: var(--text-dim);
    }

    .stats strong {
        color: var(--text);
    }

    .sep {
        color: var(--border);
    }

    .stat-ready strong { color: var(--green); }
    .stat-review strong { color: var(--yellow); }
    .stat-done strong { color: var(--green); }
    .stat-error strong { color: var(--red); }

    .actions {
        display: flex;
        gap: 8px;
    }

    .btn-reset, .btn-open {
        padding: 6px 16px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: transparent;
        color: var(--text-dim);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.15s;
    }

    .btn-reset:hover, .btn-open:hover {
        border-color: var(--text-dim);
        color: var(--text);
    }

    .btn-open {
        border-color: var(--green);
        color: var(--green);
    }

    .btn-open:hover {
        background: rgba(52, 211, 153, 0.1);
    }

    .btn-apply {
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

    .btn-apply:hover:not(:disabled) {
        background: var(--accent-hover);
    }

    .btn-apply:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

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
        width: 340px;
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .choice-title {
        margin: 0 0 4px;
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        text-align: center;
    }

    .choice-btn {
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

    .choice-btn:hover {
        border-color: var(--accent);
        background: rgba(99, 102, 241, 0.05);
    }

    .choice-icon {
        font-size: 18px;
        margin-bottom: 2px;
    }

    .choice-label {
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
    }

    .choice-desc {
        font-size: 12px;
        color: var(--text-dim);
    }
</style>
