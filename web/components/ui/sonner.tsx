'use client'

import type { CSSProperties } from 'react'
import { Toaster as Sonner, ToasterProps } from 'sonner'
import { useThemeMode } from '@/components/theme-provider'

const Toaster = ({ ...props }: ToasterProps) => {
  const { resolvedTheme } = useThemeMode()

  return (
    <Sonner
      theme={resolvedTheme as ToasterProps['theme']}
      className="toaster group"
      style={
        {
          '--normal-bg': 'var(--popover)',
          '--normal-text': 'var(--popover-foreground)',
          '--normal-border': 'var(--border)',
        } as CSSProperties
      }
      {...props}
    />
  )
}

export { Toaster }
