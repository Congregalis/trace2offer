# Discovery Onboarding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add first-run discovery onboarding so new users can add recommended discovery rules and understand the fields without leaving the product.

**Architecture:** Keep the current two-page product shape and existing visual system. Implement onboarding as shared preset data plus two UI entry points: a candidate-pool empty-state quick-start area and a persistent example/help area inside the discovery rules dialog. Drive both UI surfaces and the repository documentation from one preset/copy source to avoid drift.

**Tech Stack:** Next.js App Router, React client components, Zustand stores, existing shadcn/ui components, Markdown docs

---

## File Structure

- Create: `docs/discovery-rules-quickstart.md`
- Create: `web/lib/discovery-presets.ts`
- Create: `web/components/discovery-preset-cards.tsx`
- Create: `web/components/discovery-quickstart-dialog.tsx`
- Modify: `web/components/candidates-table.tsx`
- Modify: `web/components/discovery-rules-panel.tsx`
- Reuse: `web/lib/discovery-store.ts`
- Reuse: `web/lib/types.ts`

Notes:

- Do not introduce a new page or separate theme system.
- Reuse existing `Dialog`, `Button`, `Badge`, `Input`, `Textarea`, `Table`.
- Keep the existing muted/card styling semantics.
- This repo does not currently have a frontend unit-test runner. For this feature, use `pnpm build` plus browser smoke checks as the verification baseline rather than inventing a new test harness mid-feature.

### Task 1: Shared Preset Data And Quickstart Doc

**Files:**
- Create: `docs/discovery-rules-quickstart.md`
- Create: `web/lib/discovery-presets.ts`
- Modify: `web/lib/types.ts` only if a supporting type is still missing

- [ ] **Step 1: Write the quickstart doc outline before implementation**

Document these sections in `docs/discovery-rules-quickstart.md`:

```md
# Discovery Rules Quickstart

## What Discovery Rules Do
## Where To Find RSS/Atom Feeds
## Recommended Starter Feeds
## How To Fill Each Field
## Recommended Keywords
## Common Mistakes
```

- [ ] **Step 2: Add shared preset data module**

Create `web/lib/discovery-presets.ts` with:

```ts
export type DiscoveryPresetGroup = "priority" | "general";

export interface DiscoveryPreset {
  id: string;
  name: string;
  summary: string;
  group: DiscoveryPresetGroup;
  tags: string[];
  rule: DiscoveryRuleMutationInput;
}

export const DISCOVERY_PRESETS: DiscoveryPreset[] = [/* five approved presets */];
export function hasMatchingRule(preset: DiscoveryPreset, rules: DiscoveryRule[]): boolean { /* compare by feedUrl or name */ }
```
```

- [ ] **Step 3: Run build to catch type or import mistakes**

Run: `cd web && pnpm build`  
Expected: build succeeds with the new shared preset module and docs present

- [ ] **Step 4: Commit**

```bash
git add docs/discovery-rules-quickstart.md web/lib/discovery-presets.ts web/lib/types.ts
git commit -m "feat(web): add discovery onboarding presets and quickstart doc"
```

### Task 2: Candidate Empty-State Quick Start

**Files:**
- Create: `web/components/discovery-preset-cards.tsx`
- Modify: `web/components/candidates-table.tsx`
- Reuse: `web/lib/discovery-store.ts`

- [ ] **Step 1: Add a reusable preset card list component**

Create `web/components/discovery-preset-cards.tsx` that:

- accepts a list of presets
- renders grouped cards
- shows `一键添加` or `已添加`
- exposes callbacks for add/help actions

- [ ] **Step 2: Implement candidate-pool empty-state onboarding**

In `web/components/candidates-table.tsx`:

- read discovery rules from `useDiscoveryStore`
- show the quick-start block only when `rules.length === 0`
- render the `priority` and `general` preset groups
- keep existing page styling and spacing conventions

- [ ] **Step 3: Wire one-click add and refresh**

For each preset card:

- call `addRule(preset.rule)`
- refresh discovery rules
- allow the user to run discovery next
- disable cards already added using `hasMatchingRule`

- [ ] **Step 4: Verify**

Run: `cd web && pnpm build`  
Expected: build succeeds

Manual smoke:

1. Open the candidate tab with no discovery rules.
2. Confirm the quick-start block appears.
3. Click one preset.
4. Confirm the rule is created and the card changes to `已添加`.

- [ ] **Step 5: Commit**

```bash
git add web/components/discovery-preset-cards.tsx web/components/candidates-table.tsx
git commit -m "feat(web): add candidate empty-state discovery quickstart"
```

### Task 3: Discovery Dialog Persistent Examples And Help

**Files:**
- Create: `web/components/discovery-quickstart-dialog.tsx`
- Modify: `web/components/discovery-rules-panel.tsx`
- Reuse: `web/components/discovery-preset-cards.tsx`
- Reuse: `docs/discovery-rules-quickstart.md` as copy source

- [ ] **Step 1: Add a quickstart help dialog component**

Create `web/components/discovery-quickstart-dialog.tsx` that presents:

- short explanation of RSS/Atom
- starter feeds
- field-by-field guidance
- keyword examples
- link or note pointing to the repository doc content

- [ ] **Step 2: Add persistent help and preset sections to discovery rules dialog**

Modify `web/components/discovery-rules-panel.tsx` to:

- add `不会填？看快速上手`
- render preset cards even when rules already exist
- keep form layout and current theme intact

- [ ] **Step 3: Reuse the same add flow inside the dialog**

Ensure the dialog preset list:

- uses the same `DISCOVERY_PRESETS`
- uses the same duplicate detection
- updates the rule list after add
- does not fork copy from the empty-state view

- [ ] **Step 4: Verify**

Run: `cd web && pnpm build`  
Expected: build succeeds

Manual smoke:

1. Open `发现规则管理`.
2. Confirm `不会填？看快速上手` is visible.
3. Confirm recommended presets are visible regardless of existing rules.
4. Add a preset from inside the dialog.
5. Confirm duplicate presets show as `已添加`.

- [ ] **Step 5: Commit**

```bash
git add web/components/discovery-quickstart-dialog.tsx web/components/discovery-rules-panel.tsx web/components/discovery-preset-cards.tsx
git commit -m "feat(web): add persistent discovery examples and quickstart help"
```

### Task 4: Final Verification And Fit Check

**Files:**
- Modify only if needed to fix verification findings

- [ ] **Step 1: Run final build**

Run: `cd web && pnpm build`  
Expected: production build succeeds

- [ ] **Step 2: Run backend regression if any shared contracts changed**

Run: `cd backend && go test ./...`  
Expected: all Go tests pass

- [ ] **Step 3: Manual UI fit check**

Verify:

1. The onboarding surfaces use the existing visual system.
2. No new page was introduced.
3. Empty-state onboarding disappears once rules exist.
4. The dialog still works for experienced users.

- [ ] **Step 4: Commit any final fit fixes**

```bash
git add web backend
git commit -m "fix(web): polish discovery onboarding integration"
```
