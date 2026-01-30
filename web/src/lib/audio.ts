export const playSfx = (audio: HTMLAudioElement | null, muted: boolean) => {
    if (!audio || muted) return;

    // Reset and play
    audio.currentTime = 0;
    audio.play().catch(e => {
        // If it's a NotSupportedError or similar, it usually means the file is missing or blocked by autoplay
        if (e.name === 'NotSupportedError') {
            console.warn(`SFX Asset at ${audio.src} could not be loaded. It may be missing from public folder.`);
        } else if (e.name === 'NotAllowedError') {
            console.warn("SFX playback blocked by browser autocomplete policy. User interaction required.");
        } else {
            console.error("SFX play failed:", e);
        }
    });
};
