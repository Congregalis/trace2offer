"use client";

import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { PrepScope } from "@/lib/prep-types";

const SCOPE_LABELS: Record<PrepScope, string> = {
  topics: "Topic",
};

export interface KnowledgeScopeSwitcherProps {
  scopes: PrepScope[];
  value: PrepScope;
  onChange: (scope: PrepScope) => void;
  disabled?: boolean;
}

export function KnowledgeScopeSwitcher({ scopes, value, onChange, disabled = false }: KnowledgeScopeSwitcherProps) {
  return (
    <Tabs
      value={value}
      onValueChange={(nextValue) => {
        const nextScope = nextValue as PrepScope;
        if (!scopes.includes(nextScope)) {
          return;
        }
        onChange(nextScope);
      }}
    >
      <TabsList className="w-full">
        {scopes.map((scope) => (
          <TabsTrigger key={scope} value={scope} disabled={disabled}>
            {SCOPE_LABELS[scope]}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
