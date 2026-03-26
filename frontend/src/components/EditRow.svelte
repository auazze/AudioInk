<script>
    import { createEventDispatcher } from 'svelte';

    export let file;
    export let index;
    export let showStatus = false;
    export let pendingArtist = '';
    export let pendingTitle = '';

    const dispatch = createEventDispatcher();

    let editingField = null;
    let editValue = '';

    // Live diff: does the pending bulk value differ from the current?
    $: artistDiff = pendingArtist && pendingArtist !== file.artist;
    $: titleDiff = pendingTitle && pendingTitle !== file.title;
    $: hasDiff = artistDiff || titleDiff;

    // Preview filename when pending changes exist
    $: previewFilename = hasDiff ? buildFilename(
        artistDiff ? pendingArtist : file.artist,
        titleDiff ? pendingTitle : file.title,
        file.extras,
        file.filename
    ) : '';
    $: filenameDiff = hasDiff && previewFilename && previewFilename !== file.filename;

    function buildFilename(artist, title, extras, original) {
        const ext = original.substring(original.lastIndexOf('.'));
        let name = '';
        if (artist && title) name = artist + ' - ' + title;
        else if (title) name = title;
        if (!name) return original;
        if (extras) name += ' (' + extras + ')';
        return name + ext;
    }

    function startEdit(field) {
        if (showStatus) return;
        editingField = field;
        editValue = file[field];
    }

    function commitEdit() {
        if (editingField && editValue !== file[editingField]) {
            dispatch('update', { index, field: editingField, value: editValue });
        }
        editingField = null;
    }

    function handleKeydown(e) {
        if (e.key === 'Enter') commitEdit();
        if (e.key === 'Escape') { editingField = null; }
    }

    function statusIcon(file) {
        if (file.status === 'done') return '\u2713';
        if (file.status === 'error') return '!';
        return '';
    }

    function statusClass(file) {
        if (file.status === 'done') return 'stat-done';
        if (file.status === 'error') return 'stat-error';
        return '';
    }

    function confidenceClass(c) {
        if (c === 'high') return 'conf-high';
        if (c === 'medium') return 'conf-medium';
        return 'conf-low';
    }

    function confidenceIcon(c) {
        if (c === 'high') return '\u2713';
        if (c === 'medium') return '~';
        return '?';
    }
</script>

<tr class="file-row" class:row-done={file.status === 'done'} class:row-error={file.status === 'error'} class:row-preview={hasDiff}>
    <!-- Filename -->
    <td class="cell-filename" title={showStatus && file.outputFilename ? file.outputFilename : file.filename}>
        {#if showStatus && file.outputFilename}
            <span class="new-name">{file.outputFilename}</span>
        {:else if filenameDiff}
            <span class="diff-old">{file.filename}</span>
            <span class="diff-new">{previewFilename}</span>
        {:else}
            {file.filename}
        {/if}
    </td>

    <!-- Artist -->
    <td class="cell-editable" on:dblclick={() => startEdit('artist')}>
        {#if editingField === 'artist'}
            <input class="edit-input" bind:value={editValue} on:blur={commitEdit} on:keydown={handleKeydown} autofocus />
        {:else if artistDiff}
            <span class="diff-old">{file.artist || '\u2014'}</span>
            <span class="diff-new">{pendingArtist}</span>
        {:else}
            <span class="cell-value" class:empty={!file.artist}>{file.artist || '\u2014'}</span>
        {/if}
    </td>

    <!-- Title -->
    <td class="cell-editable" on:dblclick={() => startEdit('title')}>
        {#if editingField === 'title'}
            <input class="edit-input" bind:value={editValue} on:blur={commitEdit} on:keydown={handleKeydown} autofocus />
        {:else if titleDiff}
            <span class="diff-old">{file.title || '\u2014'}</span>
            <span class="diff-new">{pendingTitle}</span>
        {:else}
            <span class="cell-value" class:empty={!file.title}>{file.title || '\u2014'}</span>
        {/if}
    </td>

    <!-- Confidence / Status -->
    <td class="cell-confidence">
        {#if showStatus}
            <span class="conf-badge {statusClass(file)}" title={file.statusError || 'OK'}>
                {statusIcon(file)}
            </span>
        {:else}
            <span class="conf-badge {confidenceClass(file.confidence)}" title="{file.confidence} confidence">
                {confidenceIcon(file.confidence)}
            </span>
        {/if}
    </td>
</tr>

<style>
    .file-row {
        border-bottom: 1px solid var(--border);
        transition: background 0.15s;
    }

    .file-row:hover { background: var(--surface-hover); }
    .row-done { background: rgba(52, 211, 153, 0.03); }
    .row-error { background: rgba(248, 113, 113, 0.05); }
    .row-preview { background: rgba(99, 102, 241, 0.03); }

    td {
        padding: 6px 12px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 250px;
        vertical-align: middle;
    }

    .cell-filename {
        color: var(--text-dim);
        font-size: 12px;
    }

    .new-name { color: var(--green); font-size: 12px; }

    .cell-editable { cursor: text; }
    .cell-value.empty { color: var(--text-dim); font-style: italic; }

    .edit-input {
        width: 100%;
        padding: 2px 6px;
        background: var(--bg);
        border: 1px solid var(--accent);
        border-radius: 4px;
        color: var(--text);
        font-size: 13px;
        outline: none;
    }

    /* Diff preview */
    .diff-old {
        display: block;
        font-size: 11px;
        color: var(--red, #f87171);
        text-decoration: line-through;
        opacity: 0.7;
        line-height: 1.3;
    }

    .diff-new {
        display: block;
        font-size: 12px;
        color: var(--green, #34d399);
        line-height: 1.3;
    }

    .cell-confidence { text-align: center; width: 40px; }

    .conf-badge {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 22px;
        height: 22px;
        border-radius: 50%;
        font-size: 12px;
        font-weight: 600;
    }

    .conf-high, .stat-done { background: rgba(52, 211, 153, 0.15); color: var(--green); }
    .conf-medium { background: rgba(251, 191, 36, 0.15); color: var(--yellow); }
    .conf-low, .stat-error { background: rgba(248, 113, 113, 0.15); color: var(--red); }
</style>
