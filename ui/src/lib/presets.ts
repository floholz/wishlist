import { t } from './i18n/index.svelte'

// List presets ("templates"). Names and default titles live in the language files (presets.*). Swap the images in ui/public/presets/ for real photos —
// keep the file names or adjust `image` here. `accent` tints buttons, numbers and the
// page background of a list using that preset.
export type PresetId = 'blank' | 'christmas' | 'birthday' | 'wedding' | 'baby'

export interface Preset {
  id: PresetId
  image: string
  accent: string
}

export const PRESETS: Preset[] = [
  { id: 'blank', image: '/presets/blank.svg', accent: '#57534e' },
  { id: 'christmas', image: '/presets/christmas.svg', accent: '#b4232c' },
  { id: 'birthday', image: '/presets/birthday.svg', accent: '#d9467a' },
  { id: 'wedding', image: '/presets/wedding.svg', accent: '#a07a45' },
  { id: 'baby', image: '/presets/baby.svg', accent: '#3f8f86' },
]

/** Translated name ("Christmas") and default list title ("Christmas wishlist"). */
export const presetLabel = (id: PresetId) => t(`presets.${id}.label`)
export const presetTitle = (id: PresetId) => t(`presets.${id}.title`)

export const preset = (id: string | undefined): Preset => PRESETS.find(p => p.id === id) ?? PRESETS[0]

/** Inline style that themes a subtree with the preset's colours. */
export const presetStyle = (id: string | undefined) => `--accent: ${preset(id).accent};`
