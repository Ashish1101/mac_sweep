import { writable } from 'svelte/store';

const STORAGE_KEY = 'macsweep-settings';

const defaults = {
  theme: 'dark',
  refreshInterval: 2,
  alwaysUseTrash: true,
  requireConfirmation: true,
  trashRetention: 30,
  operationLogging: true,
  deleteSound: 'default',
  restoreSound: 'default',
};

function loadSettings() {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      return { ...defaults, ...JSON.parse(stored) };
    }
  } catch {}
  return { ...defaults };
}

function createSettingsStore() {
  const { subscribe, set, update } = writable(loadSettings());

  subscribe(value => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
    } catch {}
  });

  return {
    subscribe,
    set,
    update,
    setSetting(key, value) {
      update(s => ({ ...s, [key]: value }));
    },
  };
}

export const settings = createSettingsStore();
