"use client";

import { Clock3, MoonStar, SunMedium } from "lucide-react";
import { useThemeMode, type ThemeMode } from "@/components/theme-provider";
import { cn } from "@/lib/utils";

const THEME_OPTIONS: Array<{
  value: ThemeMode;
  label: string;
  icon: typeof Clock3;
}> = [
  { value: "auto", label: "自动", icon: Clock3 },
  { value: "light", label: "亮色", icon: SunMedium },
  { value: "dark", label: "暗色", icon: MoonStar },
];

export function ThemeToggle() {
  const { mode, setMode } = useThemeMode();
  const activeIndex = Math.max(
    THEME_OPTIONS.findIndex((option) => option.value === mode),
    0
  );

  return (
    <div className="relative inline-grid grid-cols-3 items-center rounded-full border border-border/70 bg-background/65 p-1 shadow-[var(--panel-shadow)] backdrop-blur-xl">
      <span
        aria-hidden
        className="pointer-events-none absolute inset-y-1 left-1 w-[calc((100%-0.5rem)/3)] rounded-full bg-foreground shadow-[var(--panel-shadow-strong)] transition-transform duration-300 ease-[var(--ease-fluid)]"
        style={{ transform: `translateX(${activeIndex * 100}%)` }}
      />
      {THEME_OPTIONS.map((option) => {
        const active = mode === option.value;
        const Icon = option.icon;

        return (
          <button
            key={option.value}
            type="button"
            onClick={() => setMode(option.value)}
            aria-pressed={active}
            aria-label={`切换到${option.label}主题`}
            title={option.label}
            className={cn(
              "relative z-10 inline-flex h-8 w-8 items-center justify-center rounded-full transition-colors duration-300 ease-[var(--ease-fluid)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
              active ? "text-background" : "text-muted-foreground hover:text-foreground"
            )}
          >
            <Icon className="h-4 w-4" />
          </button>
        );
      })}
    </div>
  );
}
