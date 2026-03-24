import type { Metadata, Viewport } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import { Analytics } from '@vercel/analytics/next'
import { ThemeProvider } from '@/components/theme-provider'
import { Toaster } from '@/components/ui/sonner'
import './globals.css'

const _geist = Geist({ subsets: ["latin"] });
const _geistMono = Geist_Mono({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: 'Trace2Offer - 求职线索台',
  description: '用一个极简看板集中管理职位/公司线索与跟进状态',
}

export const viewport: Viewport = {
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#fff7eb' },
    { media: '(prefers-color-scheme: dark)', color: '#141b2f' },
  ],
}

const themeInitScript = `
(() => {
  const MODE_KEY = 'trace2offer-theme-mode';
  const RESOLVED_KEY = 'trace2offer-theme-resolved';
  const DAY_THEME_START_HOUR = 7;
  const NIGHT_THEME_START_HOUR = 19;
  const root = document.documentElement;
  const isMode = (value) => value === 'auto' || value === 'light' || value === 'dark';
  const isResolved = (value) => value === 'light' || value === 'dark';
  const readStorage = (key) => {
    try {
      return window.localStorage.getItem(key);
    } catch {
      return null;
    }
  };
  const resolveThemeByTime = () => {
    const hour = new Date().getHours();
    return hour >= DAY_THEME_START_HOUR && hour < NIGHT_THEME_START_HOUR ? 'light' : 'dark';
  };
  const storedMode = readStorage(MODE_KEY);
  const mode = isMode(storedMode) ? storedMode : 'auto';
  const resolvedTheme = mode === 'auto' ? resolveThemeByTime() : mode;
  root.dataset.themeMode = mode;
  root.dataset.resolvedTheme = resolvedTheme;
  root.classList.toggle('dark', resolvedTheme === 'dark');
  root.style.colorScheme = resolvedTheme;
})();
`

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body className="font-sans antialiased">
        <ThemeProvider>
          {children}
          <Toaster />
          <Analytics />
        </ThemeProvider>
      </body>
    </html>
  )
}
