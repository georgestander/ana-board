const ROWS = 6;
const COLS = 22;
const MAX_ANIMATION_DURATION_MS = 3000;
const TILE_FLIP_DURATION_MS = 180;
const NATURAL_TILE_STEP_MS = 70;
const FLAP_SAMPLE_URL = "/static/sounds/split-flap.wav?v=6";
const board = document.querySelector("#board");
const connection = document.querySelector("#connection");
const soundToggle = document.querySelector("#sound-toggle");
const TILE_COLOR_CLASSES = ["color-white", "color-green", "color-amber", "color-red", "color-blue", "color-violet"];
const tiles = [];
let current = blankCells();
let currentColors = blankColors();
let flapAudio = null;
let soundEnabled = false;

if (new URLSearchParams(window.location.search).has("embed")) {
  document.body.classList.add("is-embedded");
}

function blankCells() {
  return Array.from({ length: ROWS }, () => Array.from({ length: COLS }, () => " "));
}

function blankColors() {
  return Array.from({ length: ROWS }, () => Array.from({ length: COLS }, () => "white"));
}

function createBoard() {
  board.textContent = "";
  for (let row = 0; row < ROWS; row += 1) {
    tiles[row] = [];
    for (let col = 0; col < COLS; col += 1) {
      const tile = document.createElement("div");
      tile.className = "tile is-space";
      tile.textContent = " ";
      tile.dataset.row = row;
      tile.dataset.col = col;
      board.appendChild(tile);
      tiles[row][col] = tile;
    }
  }
}

function setupSoundToggle() {
  if (!soundToggle) {
    return;
  }

  if (!window.Audio) {
    soundToggle.textContent = "no sound";
    soundToggle.disabled = true;
    return;
  }

  soundToggle.addEventListener("click", async () => {
    soundEnabled = !soundEnabled;
    soundToggle.classList.toggle("is-on", soundEnabled);
    soundToggle.setAttribute("aria-pressed", String(soundEnabled));
    soundToggle.textContent = soundEnabled ? "sound on" : "sound off";

    if (!soundEnabled) {
      return;
    }

    ensureFlapAudio().load();
  });
}

async function loadCurrentFrame() {
  const response = await fetch("/api/current", { headers: { Accept: "application/json" } });
  if (!response.ok) {
    throw new Error(`current frame request failed: ${response.status}`);
  }

  const payload = await response.json();
  applyFrame(payload.frame);
}

function connectEvents() {
  const events = new EventSource("/events");

  events.addEventListener("open", () => {
    connection.textContent = "live";
    connection.classList.add("is-live");
  });

  events.addEventListener("error", () => {
    connection.textContent = "reconnecting";
    connection.classList.remove("is-live");
  });

  events.addEventListener("frame", (event) => {
    const payload = JSON.parse(event.data);
    applyFrame(payload.frame);
  });
}

function applyFrame(frame) {
  if (!frame || !Array.isArray(frame.cells)) {
    return;
  }

  const next = frame.cells;
  const nextColors = normalizeColors(frame.colors);
  const changes = collectChanges(next, nextColors);
  const ordered = orderChanges(changes);
  const timing = animationTiming(ordered.length);

  playRowSound(changes.length);

  ordered.forEach((change, index) => {
    const delay = timing.stepDelay * index;
    window.setTimeout(() => updateTile(change.row, change.col, change.char, change.color, timing.flipDuration), delay);
  });

  current = next.map((row) => row.slice());
  currentColors = nextColors.map((row) => row.slice());
}

function animationTiming(changeCount) {
  if (changeCount === 0) {
    return { stepDelay: 0, flipDuration: TILE_FLIP_DURATION_MS };
  }

  if (changeCount === 1) {
    return { stepDelay: 0, flipDuration: TILE_FLIP_DURATION_MS };
  }

  const naturalDuration = TILE_FLIP_DURATION_MS + (changeCount - 1) * NATURAL_TILE_STEP_MS;
  const targetDuration = Math.min(MAX_ANIMATION_DURATION_MS, naturalDuration);
  const availableWindow = targetDuration - TILE_FLIP_DURATION_MS;
  return {
    stepDelay: availableWindow / (changeCount - 1),
    flipDuration: TILE_FLIP_DURATION_MS,
  };
}

function normalizeColors(colors) {
  if (!Array.isArray(colors)) {
    return blankColors();
  }

  return Array.from({ length: ROWS }, (_, row) =>
    Array.from({ length: COLS }, (_, col) => sanitizeColor(colors[row]?.[col] || "white")),
  );
}

function collectChanges(next, nextColors) {
  const changes = [];
  for (let row = 0; row < ROWS; row += 1) {
    for (let col = 0; col < COLS; col += 1) {
      const char = next[row]?.[col] || " ";
      const color = nextColors[row][col];
      if (char !== current[row][col] || color !== currentColors[row][col]) {
        changes.push({ row, col, char, color });
      }
    }
  }
  return changes;
}

function orderChanges(changes) {
  return changes;
}

function updateTile(row, col, char, color, flipDuration) {
  const tile = tiles[row]?.[col];
  if (!tile) {
    return;
  }

  tile.classList.remove("is-changing");
  TILE_COLOR_CLASSES.forEach((className) => tile.classList.remove(className));
  void tile.offsetWidth;
  tile.style.setProperty("--flap-duration", `${flipDuration}ms`);
  tile.textContent = char === " " ? " " : char;
  tile.classList.toggle("is-space", char === " ");
  tile.classList.toggle("is-emoji", isEmojiSymbol(char));
  tile.classList.add(`color-${sanitizeColor(color)}`);
  tile.classList.add("is-changing");
}

function sanitizeColor(color) {
  const normalized = String(color || "white").toLowerCase();
  if (["white", "green", "amber", "red", "blue", "violet"].includes(normalized)) {
    return normalized;
  }

  return "white";
}

function isEmojiSymbol(symbol) {
  return symbol.length > 1 || /\p{Extended_Pictographic}/u.test(symbol);
}

function ensureFlapAudio() {
  if (flapAudio) {
    return flapAudio;
  }

  flapAudio = new Audio(FLAP_SAMPLE_URL);
  flapAudio.preload = "auto";
  flapAudio.volume = 1;
  return flapAudio;
}

function playRowSound(changeCount) {
  if (!soundEnabled || changeCount === 0) {
    return;
  }

  const audio = ensureFlapAudio();
  audio.pause();
  audio.currentTime = 0;
  audio.volume = 1;
  audio.playbackRate = 1;
  audio
    .play()
    .catch((error) => {
      console.error(error);
    });
}

createBoard();
setupSoundToggle();
loadCurrentFrame()
  .then(connectEvents)
  .catch((error) => {
    connection.textContent = "offline";
    connection.classList.remove("is-live");
    console.error(error);
  });

window.anaBoardSoundState = () => ({
  enabled: soundEnabled,
  sample: FLAP_SAMPLE_URL,
  ready: Boolean(flapAudio),
  duration: flapAudio?.duration || null,
  maxAnimationDurationMs: MAX_ANIMATION_DURATION_MS,
});
