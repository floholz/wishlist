import type en from './en'

/** Plural forms keyed by Intl.PluralRules categories; `other` is always required. */
export type Plural = Partial<Record<Intl.LDMLPluralRule, string>> & { other: string }

/**
 * The shape every language must provide: same keys as English, any wording.
 * A missing or misspelled key fails `npm run check`.
 */
export type Messages = Shape<typeof en>
type Shape<T> = { [K in keyof T]: T[K] extends string ? string : T[K] extends { other: string } ? Plural : Shape<T[K]> }

/** All message keys as dotted paths, e.g. "dashboard.startNew". */
export type Key = Leaves<typeof en>
type Leaves<T, P extends string = ''> = {
  [K in keyof T & string]: T[K] extends string | { other: string } ? `${P}${K}` : Leaves<T[K], `${P}${K}.`>
}[keyof T & string]
