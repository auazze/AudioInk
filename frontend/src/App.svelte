<script>
    import DropZone from './components/DropZone.svelte';
    import FileTable from './components/FileTable.svelte';
    import ConfirmView from './components/ConfirmView.svelte';
    import ModeChooser from './components/ModeChooser.svelte';
    import { SelectFiles, SelectDirectory, ScanFiles, ApplyTagsCopy, ApplyTagsOverwrite, OpenOutputFolder, IsConfirmMode, IsChooserMode, GetInitialFiles, UndoLast, RedoLast } from '../wailsjs/go/main/App.js';
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
    let bulkArtist = '';
    let bulkTitle = '';
    let undoMessage = '';
    let isDraggingOver = false;
    let dragLeaveTimer;

    // ---- Undo / Redo ----
    // Snapshot stack of ALL actions: Set All, cell edit, drop, select,
    // clear, apply. Each entry is a deep copy of the files array plus the
    // surrounding UI state. Apply entries also carry diskWritten=true so
    // their undo additionally calls backend UndoLast to revert the
    // on-disk rename + tag write.
    let undoStack = [];
    let redoStack = [];
    const MAX_UNDO = 50;

    function captureState(kind) {
        return {
            kind,
            files: JSON.parse(JSON.stringify(files)),
            done,
            applyMode,
            diskWritten: false,
        };
    }

    // Push the CURRENT state onto undoStack BEFORE mutating it.
    // Any new action invalidates the redo stack.
    //
    // Note: Svelte 3 reactivity only triggers on REASSIGNMENT. Bare
    // `.push()` mutates the array but `canUndo` (a $: derived var) would
    // never recompute. Every undoStack/redoStack mutation in this file
    // must be followed by `stack = stack` or use spread reassignment.
    function snapshot(kind) {
        undoStack.push(captureState(kind));
        if (undoStack.length > MAX_UNDO) undoStack.shift();
        undoStack = undoStack;
        redoStack = [];
    }

    $: readyCount = files.filter(f => f.confidence === 'high' || f.confidence === 'medium').length;
    $: reviewCount = files.filter(f => f.confidence === 'low').length;
    $: canUndo = undoStack.length > 0;
    $: canRedo = redoStack.length > 0;

    onMount(async () => {
        chooserMode = await IsChooserMode();
        if (chooserMode) return;

        confirmMode = await IsConfirmMode();
        if (confirmMode) return;

        OnFileDrop((x, y, paths) => {
            isDraggingOver = false;
            handleDroppedPaths(paths);
        }, true);

        // Visual cue while a file is being dragged over the window.
        // WebView still emits HTML5 drag events for external file drags
        // even when Wails handles the actual drop natively.
        // dragleave fires for child elements too, so debounce.
        const handleEnter = (e) => {
            if (!e.dataTransfer || !Array.from(e.dataTransfer.types || []).includes('Files')) return;
            clearTimeout(dragLeaveTimer);
            isDraggingOver = true;
        };
        const handleOver = (e) => {
            e.preventDefault();
            clearTimeout(dragLeaveTimer);
            isDraggingOver = true;
        };
        const handleLeave = () => {
            dragLeaveTimer = setTimeout(() => { isDraggingOver = false; }, 80);
        };
        const handleDropEnd = () => {
            clearTimeout(dragLeaveTimer);
            isDraggingOver = false;
        };
        window.addEventListener('dragenter', handleEnter);
        window.addEventListener('dragover', handleOver);
        window.addEventListener('dragleave', handleLeave);
        window.addEventListener('drop', handleDropEnd);

        // Keyboard shortcuts. Ignore when typing in an input field so
        // Ctrl+Z inside the bulk-Artist field doesn't trigger app undo.
        const isTyping = (target) => {
            if (!target) return false;
            const tag = (target.tagName || '').toLowerCase();
            return tag === 'input' || tag === 'textarea' || target.isContentEditable;
        };
        window.addEventListener('keydown', (e) => {
            if (isTyping(e.target)) return;
            const z = e.key === 'z' || e.key === 'Z' || e.key === 'я' || e.key === 'Я';
            const y = e.key === 'y' || e.key === 'Y' || e.key === 'н' || e.key === 'Н';
            if (e.ctrlKey && !e.altKey) {
                if (z && !e.shiftKey) { e.preventDefault(); handleUndo(); return; }
                if (z && e.shiftKey)  { e.preventDefault(); handleRedo(); return; }
                if (y)                { e.preventDefault(); handleRedo(); return; }
            }
        });

        // Load files passed from context menu → "Open in AudioInk"
        const initial = await GetInitialFiles();
        if (initial && initial.length > 0) {
            files = initial;
            resetState();
        }
    });

    async function handleDroppedPaths(paths) {
        try {
            const results = await ScanFiles(paths);
            if (!results || results.length === 0) return;

            // Empty list → first drop becomes the list.
            if (files.length === 0) {
                snapshot('load');
                files = results;
                resetState();
                return;
            }

            // Files already loaded → APPEND, dedupe by filePath
            // (case-insensitive to match Windows filesystem semantics).
            const existing = new Set(files.map(f => (f.filePath || '').toLowerCase()));
            const incoming = results.filter(r => !existing.has((r.filePath || '').toLowerCase()));
            if (incoming.length === 0) return; // every dropped file already in the list

            snapshot('drop');
            files = [...files, ...incoming];

            // If a previous batch was already applied, re-enable the Apply
            // button so the user can process the newly-added files. The old
            // files keep their status='done' and applyWithMode skips them.
            if (done) {
                done = false;
                appliedCount = 0;
                errorCount = 0;
                applyResults = [];
            }
        } catch (err) {
            console.error('scan error:', err);
        }
    }

    async function handleSelectFiles() {
        try {
            const results = await SelectFiles();
            if (results && results.length > 0) {
                snapshot('select');
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
                snapshot('select');
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
        // Skip if the value didn't actually change (avoids polluting the
        // undo stack with no-op edits when user clicks away from a cell).
        if (files[index] && files[index][field] === value) return;
        snapshot('edit');
        files[index] = { ...files[index], [field]: value, confidence: 'high' };
        files[index].newFilename = rebuildFilename(files[index]);
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

        // Snapshot BEFORE we mutate UI state, mark as apply so Undo
        // knows to also revert the on-disk batch via backend UndoLast.
        snapshot('apply');

        // Skip files that were already applied in a previous batch
        // (avoids re-tagging when the user added new files after a fix).
        const requests = files
            .filter(f => f.status !== 'done')
            .map(f => ({
                filePath: f.filePath,
                artist: f.artist,
                title: f.title,
                extras: f.extras || '',
                track: f.track || 0,
            }));

        if (requests.length === 0) {
            applying = false;
            done = true;
            // Nothing actually applied — drop the apply snapshot so Undo
            // doesn't try to revert a batch that never existed.
            undoStack.pop();
            undoStack = undoStack;
            return;
        }

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

            // Mark the pre-Apply snapshot as having written to disk so that
            // Undo will additionally call backend UndoLast.
            if (appliedCount > 0 && undoStack.length > 0) {
                undoStack[undoStack.length - 1].diskWritten = true;
                undoStack = undoStack;
            } else if (appliedCount === 0 && undoStack.length > 0 && undoStack[undoStack.length - 1].kind === 'apply') {
                // Apply succeeded for 0 files (all errored) — no backend
                // batch was recorded, so the snapshot would be misleading.
                undoStack.pop();
                undoStack = undoStack;
            }

            // Auto-rescan so the user can keep editing
            setTimeout(() => rescanAfterApply(), 800);
        } catch (err) {
            console.error('apply error:', err);
            errorCount = files.length;
            // Apply threw — no disk batch recorded. Drop the snapshot.
            if (undoStack.length > 0 && undoStack[undoStack.length - 1].kind === 'apply') {
                undoStack.pop();
                undoStack = undoStack;
            }
        }

        applying = false;
    }

    function openOutput() {
        if (files.length > 0) {
            OpenOutputFolder(files[0].filePath);
        }
    }

    function clearAll() {
        if (files.length === 0) return;
        snapshot('clear');
        files = [];
        resetState();
    }

    let bulkFlash = '';

    function rebuildFilename(file) {
        if (!file || !file.filename) return '';
        const dotIdx = file.filename.lastIndexOf('.');
        const ext = dotIdx >= 0 ? file.filename.substring(dotIdx) : '';
        let name;
        if (file.artist && file.title) {
            name = file.artist + ' - ' + file.title;
        } else if (file.title) {
            name = file.title;
        } else {
            // Artist set with no title, or both empty — don't fabricate a name.
            return '';
        }
        if (file.extras) name += ' (' + file.extras + ')';
        return name + ext;
    }

    function setAllArtist() {
        const val = bulkArtist.trim();
        if (!val || files.length === 0) return;
        snapshot('set-all-artist');
        for (let i = 0; i < files.length; i++) {
            files[i] = { ...files[i], artist: val, confidence: 'high' };
            files[i].newFilename = rebuildFilename(files[i]);
        }
        files = files;
        bulkArtist = ''; // clear the input — user already confirmed by clicking Set all
        showBulkFlash(`Artist "${val}" set for ${files.length} files`);
    }

    function setAllTitle() {
        const val = bulkTitle.trim();
        if (!val || files.length === 0) return;
        snapshot('set-all-title');
        for (let i = 0; i < files.length; i++) {
            files[i] = { ...files[i], title: val, confidence: 'high' };
            files[i].newFilename = rebuildFilename(files[i]);
        }
        files = files;
        bulkTitle = '';
        showBulkFlash(`Title "${val}" set for ${files.length} files`);
    }

    function showBulkFlash(msg) {
        bulkFlash = msg;
        setTimeout(() => bulkFlash = '', 2500);
    }

    async function rescanPaths(paths) {
        if (!paths || paths.length === 0) return;
        try {
            const results = await ScanFiles(paths);
            if (results && results.length > 0) {
                files = results;
                resetState();
            }
        } catch (err) {
            console.error('rescan error:', err);
        }
    }

    async function rescanAfterApply() {
        // Copy mode: originals stay untouched, copies live in AudioInk/ subfolder.
        // Rescanning would either replace the working set with copies (confusing —
        // user loses the originals view) or rescan originals (status='done' is lost
        // and the "Open output folder" button vanishes). Leave the post-apply UI
        // alone; user clicks Clear when ready, or drags more files in to append.
        if (applyMode === 'copy') return;

        const paths = applyResults
            .filter(r => r.success && (r.newPath || r.filePath))
            .map(r => r.newPath || r.filePath);
        await rescanPaths(paths);
    }

    function describeAction(kind) {
        switch (kind) {
            case 'set-all-artist': return 'Set all artist';
            case 'set-all-title':  return 'Set all title';
            case 'edit':           return 'edit';
            case 'drop':           return 'added files';
            case 'load':           return 'load';
            case 'select':         return 'select';
            case 'clear':          return 'clear';
            case 'apply':          return 'apply';
            default:               return kind || 'action';
        }
    }

    // Build a stack entry that preserves the kind/diskWritten of the
    // action it represents, so undo→redo→undo round-trips keep working
    // (especially for Apply, which has the disk-revert side effect).
    function captureCurrent(kind, diskWritten) {
        return {
            kind,
            diskWritten,
            files: JSON.parse(JSON.stringify(files)),
            done,
            applyMode,
        };
    }

    async function handleUndo() {
        // Frontend stack first — covers UI-only changes (Set All, edit,
        // drop, etc.) plus pre-Apply snapshots that also revert disk.
        if (undoStack.length > 0) {
            const snap = undoStack.pop();
            // The redo entry carries the kind+diskWritten of the action
            // we just undid, so Redo knows whether to re-call RedoLast.
            redoStack.push(captureCurrent(snap.kind, snap.diskWritten));
            if (redoStack.length > MAX_UNDO) redoStack.shift();
            undoStack = undoStack; // reactivity (see snapshot())
            redoStack = redoStack;

            if (snap.kind === 'apply' && snap.diskWritten) {
                try {
                    await UndoLast();
                } catch (err) {
                    console.warn('apply undo (disk):', err);
                }
            }

            files = snap.files;
            done = snap.done;
            applyMode = snap.applyMode;

            undoMessage = `Undo: ${describeAction(snap.kind)}`;
            setTimeout(() => undoMessage = '', 2500);
            return;
        }

        // Frontend stack empty — fall back to backend (cross-session disk
        // batches, e.g. CLI auto-fix from a prior process).
        try {
            const paths = await UndoLast();
            const n = paths ? paths.length : 0;
            undoMessage = n === 1 ? 'Undo: 1 file reverted (disk)' : `Undo: ${n} files reverted (disk)`;
            await rescanPaths(paths);
        } catch (err) {
            undoMessage = '' + (err.message || err);
        }
        setTimeout(() => undoMessage = '', 3000);
    }

    async function handleRedo() {
        if (redoStack.length > 0) {
            const snap = redoStack.pop();
            undoStack.push(captureCurrent(snap.kind, snap.diskWritten));
            if (undoStack.length > MAX_UNDO) undoStack.shift();
            undoStack = undoStack; // reactivity (see snapshot())
            redoStack = redoStack;

            if (snap.kind === 'apply' && snap.diskWritten) {
                try {
                    await RedoLast();
                } catch (err) {
                    console.warn('apply redo (disk):', err);
                }
            }

            files = snap.files;
            done = snap.done;
            applyMode = snap.applyMode;

            undoMessage = `Redo: ${describeAction(snap.kind)}`;
            setTimeout(() => undoMessage = '', 2500);
            return;
        }

        // Frontend stack empty — try backend.
        try {
            const paths = await RedoLast();
            const n = paths ? paths.length : 0;
            undoMessage = n === 1 ? 'Redo: 1 file reapplied (disk)' : `Redo: ${n} files reapplied (disk)`;
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
<div class="app" class:dragging={isDraggingOver} style="--wails-drop-target: drop">
    {#if isDraggingOver}
        <div class="drop-overlay">
            <div class="drop-overlay-inner">
                <div class="drop-overlay-icon">⤓</div>
                <div class="drop-overlay-text">
                    {files.length === 0 ? 'Drop audio files to start' : 'Drop to add more files'}
                </div>
            </div>
        </div>
    {/if}
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
                <button class="btn-undo" on:click={handleUndo} disabled={!canUndo} title="Undo last action (Ctrl+Z)">Undo</button>
                <button class="btn-undo" on:click={handleRedo} disabled={!canRedo} title="Redo (Ctrl+Y)">Redo</button>
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

    .btn-undo:hover:not(:disabled) {
        background: rgba(251, 191, 36, 0.15);
        border-color: var(--yellow, #fbbf24);
    }

    .btn-undo:disabled {
        opacity: 0.35;
        cursor: not-allowed;
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

    /* Full-window drop overlay: only visible while a file is being dragged. */
    .drop-overlay {
        position: fixed;
        inset: 0;
        z-index: 200;
        pointer-events: none;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(99, 102, 241, 0.12);
        backdrop-filter: blur(3px);
        border: 3px dashed var(--accent);
        border-radius: 4px;
        animation: dropFadeIn 0.12s ease-out;
    }

    @keyframes dropFadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }

    .drop-overlay-inner {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 10px;
        padding: 28px 40px;
        background: rgba(15, 15, 20, 0.85);
        border-radius: 14px;
        border: 1px solid rgba(99, 102, 241, 0.5);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    }

    .drop-overlay-icon {
        font-size: 40px;
        color: var(--accent);
        line-height: 1;
    }

    .drop-overlay-text {
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        letter-spacing: -0.01em;
    }

    /* Subtle outline change on the app shell during drag (in case overlay flickers). */
    .app.dragging .titlebar {
        background: rgba(99, 102, 241, 0.04);
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
