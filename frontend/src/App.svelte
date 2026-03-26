<script>
    import DropZone from './components/DropZone.svelte';
    import FileTable from './components/FileTable.svelte';
    import ManualEntryDialog from './components/ManualEntryDialog.svelte';
    import ConfirmView from './components/ConfirmView.svelte';
    import ModeChooser from './components/ModeChooser.svelte';
    import { SelectFiles, SelectDirectory, ScanFiles, ApplyTagsCopy, ApplyTagsOverwrite, ApplyQuick, OpenOutputFolder, IsConfirmMode, IsChooserMode, GetInitialFiles, UndoLast, RedoLast } from '../wailsjs/go/main/App.js';
    import { OnFileDrop } from '../wailsjs/runtime/runtime.js';
    import { onMount } from 'svelte';

    let chooserMode = false;
    let confirmMode = false;
    let files = [];
    let applying = false;
    let appliedCount = 0;
    let errorCount = 0;
    let applyResults = [];
    let done = false;
    let showChoice = false;
    let applyMode = '';
    let garbageFiles = [];
    let garbageIndex = 0;
    let showManualEntry = false;
    let bulkArtist = '';
    let bulkTitle = '';
    let undoMessage = '';

    $: readyCount = files.filter(f => f.confidence === 'high' || f.confidence === 'medium').length;
    $: reviewCount = files.filter(f => f.confidence === 'low').length;

    onMount(async () => {
        chooserMode = await IsChooserMode();
        if (chooserMode) return;

        confirmMode = await IsConfirmMode();
        if (confirmMode) return;

        OnFileDrop((x, y, paths) => {
            handleDroppedPaths(paths);
        }, true);

        // Load files passed from context menu → "Open in AudioInk"
        const initial = await GetInitialFiles();
        if (initial && initial.length > 0) {
            files = initial;
            resetState();
            checkForGarbageFiles();
        }
    });

    async function handleDroppedPaths(paths) {
        try {
            const results = await ScanFiles(paths);
            if (results && results.length > 0) {
                files = results;
                resetState();
                checkForGarbageFiles();
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
                checkForGarbageFiles();
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
                checkForGarbageFiles();
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
        garbageFiles = [];
        garbageIndex = 0;
        showManualEntry = false;
    }

    function checkForGarbageFiles() {
        garbageFiles = files
            .map((f, i) => ({ file: f, idx: i }))
            .filter(({ file }) => file.confidence === 'low');
        garbageIndex = 0;
        showManualEntry = garbageFiles.length > 0;
    }

    async function handleManualSubmit(e) {
        const { artist, title } = e.detail;
        const entry = garbageFiles[garbageIndex];
        try {
            const result = await ApplyQuick(entry.file.filePath, artist, title, '');
            files[entry.idx] = {
                ...files[entry.idx],
                artist,
                title,
                confidence: 'high',
                status: result.success ? 'done' : 'error',
                statusError: result.error || '',
                newFilename: result.newFilename || '',
                outputFilename: result.newFilename || '',
            };
            files = files;
        } catch (err) {
            console.error('ApplyQuick error:', err);
        }
        advanceGarbage();
    }

    function handleManualSkip() {
        files[garbageFiles[garbageIndex].idx] = {
            ...files[garbageFiles[garbageIndex].idx],
            skipped: true,
        };
        files = files;
        advanceGarbage();
    }

    function advanceGarbage() {
        garbageIndex++;
        if (garbageIndex >= garbageFiles.length) {
            showManualEntry = false;
            files = files.filter(f => !f.skipped);
        }
    }

    async function handleBatch(e) {
        const { useArtist, useTitle } = e.detail;
        for (let i = garbageIndex; i < garbageFiles.length; i++) {
            const entry = garbageFiles[i];
            const file = entry.file;

            let batchArtist = file.artist || '';
            let batchTitle = file.title || '';

            if (useArtist && file.cleanMetaArtist) batchArtist = file.cleanMetaArtist;
            if (useTitle && file.cleanMetaTitle) batchTitle = file.cleanMetaTitle;

            if (!batchArtist && !batchTitle) {
                files[entry.idx] = { ...files[entry.idx], skipped: true };
                continue;
            }

            try {
                const result = await ApplyQuick(file.filePath, batchArtist, batchTitle, file.extras || '');
                files[entry.idx] = {
                    ...files[entry.idx],
                    artist: batchArtist,
                    title: batchTitle,
                    confidence: 'high',
                    status: result.success ? 'done' : 'error',
                    statusError: result.error || '',
                    newFilename: result.newFilename || '',
                    outputFilename: result.newFilename || '',
                };
            } catch (err) {
                console.error('ApplyQuick batch error:', err);
            }
        }
        files = files.filter(f => !f.skipped);
        showManualEntry = false;
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

            // Auto-rescan so the user can keep editing
            setTimeout(() => rescanAfterApply(), 800);
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

    let bulkFlash = '';

    function setAllArtist() {
        if (!bulkArtist.trim()) return;
        for (let i = 0; i < files.length; i++) {
            files[i] = { ...files[i], artist: bulkArtist.trim(), confidence: 'high' };
            const f = files[i];
            const ext = f.filename.substring(f.filename.lastIndexOf('.'));
            let name = f.artist + ' - ' + (f.title || '');
            if (f.extras) name += ' (' + f.extras + ')';
            if (f.title) files[i].newFilename = name + ext;
        }
        files = files;
        showBulkFlash(`Artist "${bulkArtist.trim()}" set for ${files.length} files`);
    }

    function setAllTitle() {
        if (!bulkTitle.trim()) return;
        for (let i = 0; i < files.length; i++) {
            files[i] = { ...files[i], title: bulkTitle.trim(), confidence: 'high' };
            const f = files[i];
            const ext = f.filename.substring(f.filename.lastIndexOf('.'));
            let name = (f.artist || '') + (f.artist && f.title ? ' - ' : '') + f.title;
            if (f.extras) name += ' (' + f.extras + ')';
            if (name) files[i].newFilename = name + ext;
        }
        files = files;
        showBulkFlash(`Title "${bulkTitle.trim()}" set for ${files.length} files`);
    }

    function showBulkFlash(msg) {
        bulkFlash = msg;
        setTimeout(() => bulkFlash = '', 2500);
    }

    async function rescanPaths(paths) {
        if (!paths || paths.length === 0) return;
        const results = await ScanFiles(paths);
        if (results && results.length > 0) {
            files = results;
            resetState();
            checkForGarbageFiles();
        }
    }

    async function rescanAfterApply() {
        const paths = applyResults
            .filter(r => r.success && (r.newPath || r.filePath))
            .map(r => r.newPath || r.filePath);
        await rescanPaths(paths);
    }

    async function handleUndo() {
        try {
            const paths = await UndoLast();
            undoMessage = `Undo: ${paths.length} files reverted`;
            await rescanPaths(paths);
        } catch (err) {
            undoMessage = '' + (err.message || err);
        }
        setTimeout(() => undoMessage = '', 3000);
    }

    async function handleRedo() {
        try {
            const paths = await RedoLast();
            undoMessage = `Redo: ${paths.length} files reapplied`;
            await rescanPaths(paths);
        } catch (err) {
            undoMessage = '' + (err.message || err);
        }
        setTimeout(() => undoMessage = '', 3000);
    }
</script>

{#if chooserMode}
    <ModeChooser />
{:else if confirmMode}
    <ConfirmView />
{:else}
<div class="app" style="--wails-drop-target: drop">
    <header class="titlebar">
        <span class="title">AudioInk</span>
    </header>

    {#if files.length === 0}
        <DropZone on:selectFiles={handleSelectFiles} on:selectFolder={handleSelectFolder} />
    {:else}
        {#if !done}
            <div class="bulk-bar">
                <div class="bulk-field">
                    <span class="bulk-label">Artist:</span>
                    <input class="bulk-input" bind:value={bulkArtist} placeholder="e.g. auazze" on:keydown={e => e.key === 'Enter' && setAllArtist()} />
                    <button class="bulk-btn" on:click={setAllArtist} disabled={!bulkArtist.trim()}>Set all</button>
                </div>
                <div class="bulk-field">
                    <span class="bulk-label">Title:</span>
                    <input class="bulk-input" bind:value={bulkTitle} placeholder="e.g. Song Name" on:keydown={e => e.key === 'Enter' && setAllTitle()} />
                    <button class="bulk-btn" on:click={setAllTitle} disabled={!bulkTitle.trim()}>Set all</button>
                </div>
            </div>
        {/if}

        <FileTable {files} showStatus={done} pendingArtist={done ? '' : bulkArtist.trim()} pendingTitle={done ? '' : bulkTitle.trim()} on:update={handleUpdate} />

        {#if showManualEntry && garbageFiles[garbageIndex]}
            {#key garbageIndex}
                <ManualEntryDialog
                    file={garbageFiles[garbageIndex].file}
                    index={garbageIndex + 1}
                    total={garbageFiles.length}
                    on:submit={handleManualSubmit}
                    on:skip={handleManualSkip}
                    on:batch={handleBatch}
                />
            {/key}
        {/if}

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
                {#if undoMessage}
                    <span class="undo-msg">{undoMessage}</span>
                {/if}
                <button class="btn-undo" on:click={handleUndo} title="Undo last batch">Undo</button>
                <button class="btn-undo" on:click={handleRedo} title="Redo last undo">Redo</button>
                {#if done && applyMode === 'copy'}
                    <button class="btn-open" on:click={openOutput}>Open output folder</button>
                {/if}
                <button class="btn-reset" on:click={clearAll}>Clear</button>
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
{/if}

<style>
    .app {
        height: 100vh;
        display: flex;
        flex-direction: column;
        background: var(--bg);
    }

    .bulk-bar {
        display: flex;
        gap: 12px;
        padding: 8px 16px;
        border-bottom: 1px solid var(--border);
        background: var(--surface);
    }

    .bulk-field {
        display: flex;
        align-items: center;
        gap: 6px;
        flex: 1;
    }

    .bulk-label {
        font-size: 11px;
        color: var(--text-dim);
        flex-shrink: 0;
    }

    .bulk-input {
        flex: 1;
        padding: 5px 10px;
        background: var(--bg);
        border: 1px solid var(--border);
        border-radius: 5px;
        color: var(--text);
        font-size: 12px;
        outline: none;
    }

    .bulk-input:focus { border-color: var(--accent); }

    .bulk-btn {
        padding: 5px 10px;
        border: 1px solid var(--accent);
        border-radius: 5px;
        background: rgba(99, 102, 241, 0.1);
        color: var(--accent);
        font-size: 11px;
        cursor: pointer;
        transition: all 0.15s;
        white-space: nowrap;
    }

    .bulk-btn:hover:not(:disabled) { background: rgba(99, 102, 241, 0.2); }
    .bulk-btn:disabled { opacity: 0.4; cursor: not-allowed; }

    .bulk-flash {
        font-size: 11px;
        color: var(--green, #78b478);
        white-space: nowrap;
        flex-shrink: 0;
        animation: fadeIn 0.2s;
    }

    @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

    .btn-undo {
        padding: 6px 12px;
        border: 1px solid rgba(251, 191, 36, 0.4);
        border-radius: 6px;
        background: rgba(251, 191, 36, 0.08);
        color: var(--yellow, #fbbf24);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.15s;
    }

    .btn-undo:hover {
        background: rgba(251, 191, 36, 0.15);
        border-color: var(--yellow, #fbbf24);
    }

    .undo-msg {
        font-size: 11px;
        color: var(--green, #78b478);
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
