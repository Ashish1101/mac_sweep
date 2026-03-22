import { get } from 'svelte/store';
import { settings } from './settings.js';
import { PlaySound } from '../../wailsjs/go/main/App.js';
import faaahSrc from '../assets/sounds/faaah.mp3';
import heKnewSrc from '../assets/sounds/he-knew.mp3';
import reallyNiggaSrc from '../assets/sounds/really-nigga.mp3';
import ohMyGodSrc from '../assets/sounds/oh-my-god.mp3';
import waitAMinuteSrc from '../assets/sounds/wait-a-minute.mp3';

const customSounds = {
  faaah: faaahSrc,
  'he-knew': heKnewSrc,
  'really-nigga': reallyNiggaSrc,
  'oh-my-god': ohMyGodSrc,
  'wait-a-minute': waitAMinuteSrc,
};

// Pre-load all custom sounds into Audio elements on first import
const preloaded = {};
for (const [id, src] of Object.entries(customSounds)) {
  const audio = new Audio(src);
  audio.preload = 'auto';
  audio.load();
  preloaded[id] = audio;
}

function playCustom(id) {
  const master = preloaded[id];
  if (!master) return;
  // Clone the preloaded node so overlapping plays don't conflict
  const clone = master.cloneNode();
  clone.volume = master.volume;
  clone.play().catch(() => {});
  // Clean up after playback ends
  clone.addEventListener('ended', () => clone.remove?.(), { once: true });
}

function playByID(soundID) {
  if (customSounds[soundID]) {
    playCustom(soundID);
  } else if (soundID === 'none') {
    // silent
  } else {
    PlaySound(soundID);
  }
}

export function playDeleteSound() {
  const s = get(settings);
  playByID(s.deleteSound || 'default');
}

export function playRestoreSound() {
  const s = get(settings);
  playByID(s.restoreSound || 'default');
}

export function previewSound(soundID) {
  playByID(soundID);
}
