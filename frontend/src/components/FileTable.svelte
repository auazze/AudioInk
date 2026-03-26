<script>
    import EditRow from './EditRow.svelte';
    import { createEventDispatcher } from 'svelte';

    export let files = [];
    export let showStatus = false;
    export let pendingArtist = '';
    export let pendingTitle = '';

    const dispatch = createEventDispatcher();

    function handleUpdate(e) {
        dispatch('update', e.detail);
    }
</script>

<div class="table-wrapper">
    <table>
        <thead>
            <tr>
                <th class="col-filename">{showStatus ? 'Output' : 'Filename'}</th>
                <th class="col-artist">Artist</th>
                <th class="col-title">Title</th>
                <th class="col-conf"></th>
            </tr>
        </thead>
        <tbody>
            {#each files as file, i (file.filePath)}
                <EditRow {file} index={i} {showStatus} {pendingArtist} {pendingTitle} on:update={handleUpdate} />
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

    .col-filename { width: 35%; }
    .col-artist { width: 30%; }
    .col-title { width: 30%; }
    .col-conf { width: 5%; text-align: center; }
</style>
