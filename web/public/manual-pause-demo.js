// Manual pause/reconnect demo for the UI.
// Usage:
// 1) Run the frontend with `npm run dev`.
// 2) Open the app and make sure the GameView is visible (e.g. ?mock=game).
// 3) Open DevTools console and paste:
//    await import('/manual-pause-demo.js')
// 4) Then run: window.__ELMAKINA_DEMO__.pause(), vote(), resume(), etc.

const dev = window.__ELMAKINA_DEV__;
if (!dev) {
  throw new Error('Missing __ELMAKINA_DEV__. Open the app in dev mode first.');
}

const now = () => Date.now();

const demo = {
  pause(name = 'Yara', index = 2) {
    const deadline = now() + 60000;
    dev.pauseGame({
      paused_by_player_id: `mock-${index}`,
      paused_by_index: index,
      paused_by_name: name,
      deadline_ms: deadline,
      duration_ms: 60000,
      pause_reason: 'disconnect',
      eligible_voters: [0, 1, 3],
      kick_votes: [],
    });
  },
  vote(voterIndex = 0, voters = [0, 1, 3]) {
    dev.updateKickVotes({ eligible_voters: voters, kick_votes: [voterIndex] });
  },
  unanimous(voters = [0, 1, 3]) {
    dev.updateKickVotes({ eligible_voters: voters, kick_votes: voters });
  },
  resume() {
    dev.resumeGame();
  },
  connectionLog() {
    return dev.connectionLog();
  },
};

window.__ELMAKINA_DEMO__ = demo;
console.info('ElMakina demo helpers loaded:', Object.keys(demo));

export default demo;
