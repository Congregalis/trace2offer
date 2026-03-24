"use client";

import { Clock3, MoonStar, SunMedium } from "lucide-react";
import { useThemeMode, type ThemeMode } from "@/components/theme-provider";
import { cn } from "@/lib/utils";

const THEME_OPTIONS: Array<{
  value: ThemeMode;
  label: string;
  shortLabel: string;
  icon: typeof Clock3;
}> = [
  { value: "auto", label: "自动", shortLabel: "Auto", icon: Clock3 },
  { value: "light", label: "亮色", shortLabel: "Light", icon: SunMedium },
  { value: "dark", label: "暗色", shortLabel: "Dark", icon: MoonStar },
];

const RESOLVED_THEME_LABEL = {
  light: "日间亮色",
  dark: "夜间暗色",
} as const;

export function ThemeToggle() {
  const { mode, mounted, resolvedTheme, setMode } = useThemeMode();

  return (
    <div className="flex items-center gap-3 rounded-full border border-border/70 bg-background/65 p-1.5 pl-3 shadow-[var(--panel-shadow)] backdrop-blur-xl">
      <div className="hidden min-w-[124px] sm:block">
        <p className="text-[10px] font-semibold uppercase tracking-[0.28em] text-muted-foreground">Theme</p>
        <p className="mt-1 text-xs font-medium text-foreground">
          {mounted
            ? mode === "auto"
              ? `自动 · ${RESOLVED_THEME_LABEL[resolvedTheme]}`
              : `手动 · ${RESOLVED_THEME_LABEL[mode]}`
            : "正在同步主题"}
        </p>
      </div>

      <div className="grid grid-cols-3 gap-1 rounded-full bg-background/70 p-1">
        {THEME_OPTIONS.map((option) => {
          const active = mode === option.value;
          const Icon = option.icon;

          return (
            <button
              key={option.value}
              type="button"
              onClick={() => setMode(option.value)}
              aria-pressed={active}
              aria-label={`切换到${option.label}`}
              className={cn(
                "inline-flex h-9 items-center justify-center gap-1.5 rounded-full px-3 text-xs font-medium transition-all duration-300 ease-[var(--ease-fluid)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                active
                  ? "bg-foreground text-background shadow-[var(--panel-shadow-strong)]"
                  : "text-muted-foreground hover:bg-accent/70 hover:text-foreground"
              )}
            >
              <Icon className={cn("h-3.5 w-3.5 transition-transform duration-300", active ? "scale-100" : "scale-95")} />
              <span className="hidden md:inline">{option.label}</span>
              <span className="md:hidden">{option.shortLabel}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
