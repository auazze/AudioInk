<script>
    import EditRow from './EditRow.svelte';
    import { createEventDispatcher } from 'svelte';

    export let files = [];
    export let showStatus = false;
    export let pendingArtist = '';
    export let pendingTitle = '';
    export let selected = new Set();   // filePaths
    export let activePath = '';        // currently previewed file

    const dispatch = createEventDispatcher();

    function handleUpdate(e) { dispatch('update', e.detail); }
    function handleToggle(e) { dispatch('toggle', e.detail); }
    function handlePlay(e)   { dispatch('play', e.detail); }

    $: allSelected = files.length > 0 && files.every(f => selected.has(f.filePath));
    function toggleAll() { dispatch('toggleAll', !allSelected); }
</script>

<div class="table-wrapper">
    <table>
        <thead>
            <tr>
                <th class="col-select">
                    <input type="checkbox" checked={allSelected} on:change={toggleAll} title="Select all" />
                </th>
                <th class="col-filename">{showStatus ? 'Output' : 'Filename'}</th>
                <th class="col-artist">Artist</th>
                <th class="col-title">Title</th>
                <th class="col-health" title="Audio health"></th>
                <th class="col-conf"></th>
            </tr>
        </thead>
        <tbody>
            {#each files as file, i (file.filePath)}
                <EditRow
                    {file}
                    index={i}
                    {showStatus}
                    {pendingArtist}
                    {pendingTitle}
                    selected={selected.has(file.filePath)}
                    active={activePath === file.filePath}
                    on:update={handleUpdate}
                    on:toggle={handleToggle}
                    on:play={handlePlay}
                />
            {/each}
        </tbody>
    </table>
</div>

<style>
    .table-wrapper {
        flex: 1;
        overflow-y: auto;
        margin: 0 16px 12px;
        border: 1px solid var(--border);
        border-radius: 8px;
        background: var(--surface);
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    thead {
        position: sticky;
        top: 0;
        z-index: 1;
    }

    th {
        padding: 8px 12px;
        text-align: left;
        font-weight: 600;
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--text-dim);
        background: var(--surface);
        border-bottom: 1px solid var(--border);
    }

    .col-select { width: 7%; text-align: center; padding-left: 8px; padding-right: 0; }
    .col-select input { cursor: pointer; accent-color: var(--accent); }
    .col-filename { width: 29%; }
    .col-artist { width: 25%; }
    .col-title { width: 25%; }
    .col-health { width: 6%; text-align: center; }
    .col-conf { width: 5%; text-align: center; }
</style>
