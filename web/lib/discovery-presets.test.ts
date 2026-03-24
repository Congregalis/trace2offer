import test from "node:test";
import assert from "node:assert/strict";

import { DISCOVERY_PRESETS } from "./discovery-presets.ts";

test("discovery presets include the approved domestic community sources", () => {
  const ids = new Set(DISCOVERY_PRESETS.map((preset) => preset.id));

  assert.ok(ids.has("v2ex-jobs"), "expected V2EX jobs preset");
  assert.ok(ids.has("linuxdo-job-category"), "expected Linux.do job category preset");
  assert.ok(ids.has("linuxdo-job-tag"), "expected Linux.do job tag preset");
  assert.ok(ids.has("ruby-china-topics"), "expected Ruby China topics preset");
});

test("domestic discovery presets are explicitly tagged as domestic", () => {
  const domesticPresets = DISCOVERY_PRESETS.filter((preset) =>
    ["v2ex-jobs", "linuxdo-job-category", "linuxdo-job-tag", "ruby-china-topics"].includes(preset.id)
  );

  assert.equal(domesticPresets.length, 4, "expected all domestic presets to exist first");
  for (const preset of domesticPresets) {
    assert.ok(preset.tags.includes("国内"), `expected ${preset.id} tagged as 国内`);
  }
});
