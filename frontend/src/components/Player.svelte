<script>
    import { ComputeReplayGain } from '../../wailsjs/go/main/App.js';
    import { createEventDispatcher, onDestroy } from 'svelte';

    export let file = null; // FileResult to preview, or null

    const dispatch = createEventDispatcher();

    let audioEl;
    let ctx;            // AudioContext (created on first play — autoplay policy)
    let srcNode;        // MediaElementAudioSourceNode (created once)
    let rgGainNode;     // ReplayGain A/B
    let volGainNode;    // master volume

    let playing = false;
    let currentTime = 0;
    let duration = 0;

    // Volume defaults to 25% — instrumentals/songs can be loud and system
    // volume is usually maxed. The slider is always visible and shows the %.
    let volume = 0.25;

    let rgEnabled = false;
    let rgGainDB = null;     // cached per file
    let rgLoading = false;
    let rgFile = null;       // which file rgGainDB belongs to

    $: mediaSrc = file ? `/media?path=${encodeURIComponent(file.filePath)}` : '';

    // When the previewed file changes, reset transport + RG cache.
    $: if (file && file.filePath !== rgFile) {
        rgGainDB = null;
        rgEnabled = false;
        rgFile = null;
        currentTime = 0;
        playing = false;
    }

    function ensureGraph() {
        if (ctx) return;
        const AC = window.AudioContext || window.webkitAudioContext;
        ctx = new AC();
        srcNode = ctx.createMediaElementSource(audioEl);
        rgGainNode = ctx.createGain();
        volGainNode = ctx.createGain();
        rgGainNode.gain.value = 1;
        volGainNode.gain.value = volume;
        // source → ReplayGain → volume → speakers
        srcNode.connect(rgGainNode).connect(volGainNode).connect(ctx.destination);
    }

    async function togglePlay() {
        if (!audioEl) return;
        ensureGraph();
        if (ctx.state === 'suspended') await ctx.resume();
        if (audioEl.paused) {
            try { await audioEl.play(); } catch (e) { console.warn('play failed', e); }
        } else {
            audioEl.pause();
        }
    }

    function onTimeUpdate() { currentTime = audioEl ? audioEl.currentTime : 0; }
    function onLoaded() { duration = audioEl ? audioEl.duration || 0 : 0; }
    function onPlay() { playing = true; }
    function onPause() { playing = false; }
    function onEnded() { playing = false; currentTime = 0; }

    function seek(e) {
        if (!audioEl) return;
        audioEl.currentTime = parseFloat(e.target.value);
        currentTime = audioEl.currentTime;
    }

    // Volume slider → master gain (and a fallback if the graph isn't built yet).
    $: if (volGainNode) volGainNode.gain.value = volume;

    async function toggleRG() {
        rgEnabled = !rgEnabled;
        ensureGraph();
        if (rgEnabled && rgGainDB === null && file) {
            rgLoading = true;
            try {
                rgGainDB = await ComputeReplayGain(file.filePath);
                rgFile = file.filePath;
            } catch (e) {
                console.warn('ReplayGain failed', e);
                rgGainDB = 0;
            }
            rgLoading = false;
        }
        applyRG();
    }

    function applyRG() {
        if (!rgGainNode) return;
        const db = (rgEnabled && rgGainDB !== null) ? rgGainDB : 0;
        rgGainNode.gain.value = Math.pow(10, db / 20);
    }

    $: if (rgGainNode && rgGainDB !== null) applyRG();

    function fmt(t) {
        if (!t || isNaN(t)) return '0:00';
        const m = Math.floor(t / 60);
        const s = Math.floor(t % 60);
        return `${m}:${s.toString().padStart(2, '0')}`;
    }

    function close() { dispatch('close'); }

    onDestroy(() => {
        try { if (audioEl) audioEl.pause(); } catch (e) {}
        try { if (ctx) ctx.close(); } catch (e) {}
    });
</script>

{#if file}
<div class="player">
    <audio
        bind:this={audioEl}
        src={mediaSrc}
        on:timeupdate={onTimeUpdate}
        on:loadedmetadata={onLoaded}
        on:play={onPlay}
        on:pause={onPause}
        on:ended={onEnded}
        crossorigin="anonymous"
    ></audio>

    <button class="pl-btn pl-play" on:click={togglePlay} title={playing ? 'Pause' : 'Play'}>
        {playing ? '❚❚' : '▶'}
    </button>

    <div class="pl-name" title={file.filename}>{file.filename}</div>

    <span class="pl-time">{fmt(currentTime)}</span>
    <input
        class="pl-seek"
        type="range"
        min="0"
        max={duration || 0}
        step="0.1"
        value={currentTime}
        on:input={seek}
    />
    <span class="pl-time">{fmt(duration)}</span>

    <div class="pl-vol">
        <span class="pl-vol-icon">🔊</span>
        <input class="pl-vol-slider" type="range" min="0" max="1" step="0.01" bind:value={volume} />
        <span class="pl-vol-pct">{Math.round(volume * 100)}%</span>
    </div>

    <button
        class="pl-btn pl-rg"
        class:active={rgEnabled}
        on:click={toggleRG}
        title="Preview ReplayGain loudness (non-destructive A/B)"
    >
        {#if rgLoading}…{:else}RG{/if}
        {#if rgEnabled && rgGainDB !== null}<span class="pl-rg-db">{rgGainDB > 0 ? '+' : ''}{rgGainDB.toFixed(1)}dB</span>{/if}
    </button>

    <button class="pl-btn pl-close" on:click={close} title="Close player">✕</button>
</div>
{/if}

<style>
    .player {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 8px 16px;
        border-top: 1px solid var(--border);
        background: var(--surface);
    }

    .pl-btn {
        border: 1px solid var(--border);
        border-radius: 6px;
        background: transparent;
        color: var(--text);
        cursor: pointer;
        font-size: 12px;
        padding: 5px 9px;
        transition: all 0.15s;
        white-space: nowrap;
    }
    .pl-btn:hover { border-color: var(--accent); color: var(--accent); }

    .pl-play { min-width: 34px; }

    .pl-name {
        flex: 0 1 200px;
        font-size: 12px;
        color: var(--text-dim);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .pl-time {
        font-size: 11px;
        color: var(--text-dim);
        font-variant-numeric: tabular-nums;
        min-width: 34px;
        text-align: center;
    }

    .pl-seek { flex: 1; accent-color: var(--accent); cursor: pointer; }

    .pl-vol {
        display: flex;
        align-items: center;
        gap: 6px;
        flex-shrink: 0;
    }
    .pl-vol-icon { font-size: 12px; opacity: 0.7; }
    .pl-vol-slider { width: 80px; accent-color: var(--accent); cursor: pointer; }
    .pl-vol-pct {
        font-size: 11px;
        color: var(--text-dim);
        min-width: 32px;
        font-variant-numeric: tabular-nums;
    }

    .pl-rg { font-weight: 600; }
    .pl-rg.active { border-color: var(--green); color: var(--green); background: rgba(52, 211, 153, 0.08); }
    .pl-rg-db { font-size: 10px; margin-left: 4px; opacity: 0.85; }

    .pl-close { color: var(--text-dim); }
    .pl-close:hover { border-color: var(--red); color: var(--red); }
</style>
