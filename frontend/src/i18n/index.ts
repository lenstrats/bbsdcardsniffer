import {createContext, useContext} from 'react'
import {Messages, en} from './en'
import {de} from './de'
import {nl} from './nl'

export type {Messages} from './en'

/** Every locale the application ships. Adding one means adding a file here. */
export const locales = {en, nl, de} as const

export type Locale = keyof typeof locales

/** The order the picker lists them in. */
export const localeOrder: Locale[] = ['en', 'nl', 'de']

const storageKey = 'bbsdcardsniffer.locale'

function isLocale(v: string | null): v is Locale {
    return v !== null && v in locales
}

/** The language a first-time user gets. English rather than the system
 * language: the terminology this application deals in — partition tables,
 * filesystems, superblocks — is English in every reference a user will consult,
 * and a Dutch interface is a deliberate choice rather than an accident of which
 * machine it happens to run on. */
export const defaultLocale: Locale = 'en'

/** Resolves the locale to start in: a previous choice, otherwise the default. */
export function detectLocale(): Locale {
    try {
        const stored = localStorage.getItem(storageKey)
        if (isLocale(stored)) return stored
    } catch {
        // Private windows and blocked site data both throw; the default is fine.
    }
    return defaultLocale
}

/** Remembers a choice for next time. Failure here is not worth surfacing. */
export function storeLocale(locale: Locale): void {
    try {
        localStorage.setItem(storageKey, locale)
    } catch {
        // Ignored: the choice still applies for this run.
    }
}

export const I18nContext = createContext<Messages>(en)

/** Returns the active catalogue. */
export function useT(): Messages {
    return useContext(I18nContext)
}
