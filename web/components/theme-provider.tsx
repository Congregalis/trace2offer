'use client'

import * as React from 'react'

const THEME_MODE_STORAGE_KEY = 'trace2offer-theme-mode'
const THEME_RESOLVED_STORAGE_KEY = 'trace2offer-theme-resolved'
const DAY_THEME_START_HOUR = 7
const NIGHT_THEME_START_HOUR = 19

export type ThemeMode = 'auto' | 'light' | 'dark'
type ResolvedTheme = Exclude<ThemeMode, 'auto'>

interface ThemeContextValue {
  mode: ThemeMode
  resolvedTheme: ResolvedTheme
  mounted: boolean
  setMode: (mode: ThemeMode) => void
}

const ThemeContext = React.createContext<ThemeContextValue | null>(null)

function isThemeMode(value: string | null | undefined): value is ThemeMode {
  return value === 'auto' || value === 'light' || value === 'dark'
}

function isResolvedTheme(value: string | null | undefined): value is ResolvedTheme {
  return value === 'light' || value === 'dark'
}

function resolveThemeByTime(date = new Date()): ResolvedTheme {
  const hour = date.getHours()
  return hour >= DAY_THEME_START_HOUR && hour < NIGHT_THEME_START_HOUR ? 'light' : 'dark'
}

function safeReadStorage(key: string): string | null {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

function safeWriteStorage(key: string, value: string) {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(key, value)
  } catch {
    // Ignore storage write failures.
  }
}

function applyTheme(mode: ThemeMode, resolvedTheme: ResolvedTheme) {
  if (typeof document === 'undefined') {
    return
  }

  const root = document.documentElement
  root.dataset.themeMode = mode
  root.dataset.resolvedTheme = resolvedTheme
  root.classList.toggle('dark', resolvedTheme === 'dark')
  root.style.colorScheme = resolvedTheme
}

function readInitialMode(): ThemeMode {
  if (typeof document === 'undefined') {
    return 'auto'
  }

  const datasetMode = document.documentElement.dataset.themeMode
  if (isThemeMode(datasetMode)) {
    return datasetMode
  }

  const storedMode = safeReadStorage(THEME_MODE_STORAGE_KEY)
  return isThemeMode(storedMode) ? storedMode : 'auto'
}

function readInitialResolvedTheme(mode: ThemeMode): ResolvedTheme {
  if (typeof document === 'undefined') {
    return 'light'
  }

  const datasetTheme = document.documentElement.dataset.resolvedTheme
  if (isResolvedTheme(datasetTheme)) {
    return datasetTheme
  }

  if (mode === 'auto') {
    return resolveThemeByTime()
  }

  return mode
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setModeState] = React.useState<ThemeMode>('auto')
  const [autoTheme, setAutoTheme] = React.useState<ResolvedTheme>('light')
  const [mounted, setMounted] = React.useState(false)

  React.useEffect(() => {
    const initialMode = readInitialMode()
    const initialResolvedTheme = readInitialResolvedTheme(initialMode)

    setModeState(initialMode)
    setAutoTheme(initialResolvedTheme)
    applyTheme(initialMode, initialResolvedTheme)
    setMounted(true)
  }, [])

  React.useEffect(() => {
    if (!mounted || mode !== 'auto') {
      return
    }

    const syncThemeWithTime = () => {
      setAutoTheme(resolveThemeByTime())
    }

    syncThemeWithTime()
    const intervalId = window.setInterval(syncThemeWithTime, 60_000)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [mounted, mode])

  const resolvedTheme = mode === 'auto' ? autoTheme : mode

  React.useEffect(() => {
    if (!mounted) {
      return
    }

    applyTheme(mode, resolvedTheme)
    safeWriteStorage(THEME_MODE_STORAGE_KEY, mode)
    safeWriteStorage(THEME_RESOLVED_STORAGE_KEY, resolvedTheme)
  }, [mounted, mode, resolvedTheme])

  const setMode = React.useCallback((nextMode: ThemeMode) => {
    React.startTransition(() => {
      setModeState(nextMode)
      if (nextMode === 'auto') {
        setAutoTheme(resolveThemeByTime())
      }
    })
  }, [])

  const value = React.useMemo(
    () => ({
      mode,
      resolvedTheme,
      mounted,
      setMode,
    }),
    [mode, resolvedTheme, mounted, setMode]
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useThemeMode() {
  const context = React.useContext(ThemeContext)
  if (!context) {
    throw new Error('useThemeMode 必须在 ThemeProvider 内使用')
  }
  return context
}
