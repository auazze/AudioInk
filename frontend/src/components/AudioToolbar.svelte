<script>
    import { createEventDispatcher } from 'svelte';

    export let count = 0;          // number of files the op will act on
    export let selectedCount = 0;  // how many are explicitly selected (0 = "all")
    export let audioReady = true;
    export let busy = false;

    const dispatch = createEventDispatcher();

    const formats = [
        { label: 'MP3 320',   ext: '.mp3',  quality: '320k' },
        { label: 'MP3 VBR',   ext: '.mp3',  quality: '' },
        { label: 'FLAC',      ext: '.flac', quality: '' },
        { label: 'M4A (AAC)', ext: '.m4a',  quality: '256k' },
        { label: 'OGG',       ext: '.ogg',  quality: '' },
        { label: 'Opus',      ext: '.opus', quality: '192k' },
        { label: 'WAV',       ext: '.wav',  quality: '' },
    ];
    let fmtIndex = 0;

    $: disabled = busy || count === 0 || !audioReady;

    function convert() {
        const f = formats[fmtIndex];
        dispatch('convert', { ext: f.ext, quality: f.quality });
    }
</script>

<div class="audio-bar">
    <span class="scope" title="Operations act on selected files, or all files if none are selected">
        {selectedCount > 0 ? `${selectedCount} selected` : `all ${count}`}
    </span>

    <div class="grp">
        <select class="fmt" bind:value={fmtIndex} disabled={disabled}>
            {#each formats as f, i}
                <option value={i}>{f.label}</option>
            {/each}
        </select>
        <button class="op-btn" on:click={convert} disabled={disabled}>Convert</button>
    </div>

    <button class="op-btn" on:click={() => dispatch('normalize')} disabled={disabled} title="Even out loudness — writes a ReplayGain tag (no re-encode)">Normalize</button>
    <button class="op-btn" on:click={() => dispatch('trim')} disabled={disabled} title="Trim leading/trailing silence">Trim silence</button>
    <button class="op-btn" on:click={() => dispatch('clean')} disabled={disabled} title="Strip junk / duplicate tag blocks">Clean tags</button>
    <button class="op-btn" on:click={() => dispatch('repair')} disabled={disabled} title="Fix broken headers (lossless remux)">Repair</button>

    <span class="divider"></span>

    <button class="op-btn ghost" on:click={() => dispatch('deepScan')} disabled={busy || count === 0 || !audioReady} title="Deep health analysis (silence, clipping, transcode)">Deep scan</button>
    <button class="op-btn ghost" on:click={() => dispatch('findDupes')} disabled={busy || count === 0 || !audioReady} title="Find duplicate recordings (local, no internet)">Find duplicates</button>

    {#if !audioReady}
        <span class="warn" title="ffmpeg/ffprobe not found next to the app">⚠ audio tools unavailable</span>
    {/if}
</div>

<style>
    .audio-bar {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 7px 16px;
        border-bottom: 1px solid var(--border);
        background: var(--surface);
        flex-wrap: wrap;
    }

    .scope {
        font-size: 11px;
        color: var(--text-dim);
        min-width: 64px;
    }

    .grp { display: flex; gap: 4px; }

    .fmt {
        background: var(--bg);
        border: 1px solid var(--border);
        border-radius: 5px;
        color: var(--text);
        font-size: 11px;
        padding: 4px 6px;
        outline: none;
        cursor: pointer;
    }
    .fmt:focus { border-color: var(--accent); }

    .op-btn {
        padding: 5px 10px;
        border: 1px solid var(--accent);
        border-radius: 5px;
        background: rgba(124, 106, 239, 0.1);
        color: var(--accent);
        font-size: 11px;
        cursor: pointer;
        transition: all 0.15s;
        white-space: nowrap;
    }
    .op-btn:hover:not(:disabled) { background: rgba(124, 106, 239, 0.2); }
    .op-btn:disabled { opacity: 0.4; cursor: not-allowed; }

    .op-btn.ghost {
        border-color: var(--border);
        background: transparent;
        color: var(--text-dim);
    }
    .op-btn.ghost:hover:not(:disabled) { border-color: var(--text-dim); color: var(--text); }

    .divider { width: 1px; height: 18px; background: var(--border); margin: 0 2px; }

    .warn { font-size: 11px; color: var(--yellow); }
</style>
