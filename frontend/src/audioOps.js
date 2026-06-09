// Thin helpers around the FFmpeg-powered backend methods, kept out of
// App.svelte to respect the file-size limit. Backend bindings are regenerated
// by `wails build` / `wails dev` from the Go App methods.

import {
    ConvertFiles,
    TrimSilence,
    DetectSilence,
    CleanID3,
    RepairFiles,
    NormalizeFiles,
    FindDuplicates,
    ProbeHealth,
} from '../wailsjs/go/main/App.js';
import { EventsOn } from '../wailsjs/runtime/runtime.js';

export { CleanID3, RepairFiles, NormalizeFiles, FindDuplicates, ProbeHealth };

// subscribeAudioEvents wires the backend's streaming events. Returns an
// unsubscribe function.
export function subscribeAudioEvents(handlers) {
    const offs = [];
    const on = (ev, fn) => { if (fn) offs.push(EventsOn(ev, fn)); };
    on('health:result', handlers.onHealthResult);
    on('health:done', handlers.onHealthDone);
    on('convert:progress', handlers.onProgress);
    on('convert:done', handlers.onOpDone);
    on('trim:progress', handlers.onProgress);
    on('trim:done', handlers.onOpDone);
    on('dupes:progress', handlers.onDupesProgress);
    return () => offs.forEach(f => typeof f === 'function' && f());
}

export async function runConvert(targetPaths, ext, quality, overwrite) {
    const reqs = targetPaths.map(p => ({ filePath: p, targetExt: ext, quality }));
    return await ConvertFiles(reqs, overwrite);
}

// detectSilenceFor probes each path; returns { path: edges } for those with
// trimmable silence.
export async function detectSilenceFor(paths) {
    const map = {};
    for (const p of paths) {
        try {
            const edges = await DetectSilence(p);
            if (edges && (edges.startSec > 0.05 || edges.endSec > 0.05)) map[p] = edges;
        } catch (e) {
            console.warn('detect silence', p, e);
        }
    }
    return map;
}

export async function runTrim(edgesByPath, overwrite) {
    const reqs = Object.entries(edgesByPath).map(([filePath, edges]) => ({ filePath, edges }));
    return await TrimSilence(reqs, overwrite);
}
