// One-time move of browser data from the old "whish_*" keys (the app was briefly
// called "whish"). Imported first in main.ts, before anything reads storage.
const RENAMED = ['lists', 'visited', 'reserved', 'theme', 'lang']

try {
  for (const k of RENAMED) {
    const old = localStorage.getItem(`whish_${k}`)
    if (old !== null) {
      if (localStorage.getItem(`wishlist_${k}`) === null) localStorage.setItem(`wishlist_${k}`, old)
      localStorage.removeItem(`whish_${k}`)
    }
  }
} catch { /* storage unavailable — nothing to move */ }
