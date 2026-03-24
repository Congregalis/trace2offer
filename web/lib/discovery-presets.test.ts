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
    assert.equal(preset.group, "domestic", `expected ${preset.id} in domestic market`);
  }
});

test("international presets are grouped under overseas remote market", () => {
  const internationalPresetIds = [
    "remoteyeah-ai-engineer",
    "himalayas-remote-swe",
    "remoteyeah-backend",
    "wwr-backend",
    "smartremotejobs-swe",
    "remotefirstjobs-ai",
    "remotefirstjobs-software-dev",
    "remotefirstjobs-golang",
    "realworkfromanywhere-backend",
    "realworkfromanywhere-ai",
    "jobicy-dev-fulltime",
    "jobicy-data-science-fulltime",
    "remoteok-engineering",
    "smartremotejobs-devops",
    "smartremotejobs-data-science",
  ];

  const internationalPresets = DISCOVERY_PRESETS.filter((preset) => internationalPresetIds.includes(preset.id));
  assert.equal(internationalPresets.length, internationalPresetIds.length, "expected all international presets to exist");
  for (const preset of internationalPresets) {
    assert.equal(preset.group, "international", `expected ${preset.id} in international market`);
    assert.ok(preset.tags.includes("国外远程"), `expected ${preset.id} tagged as 国外远程`);
  }
});

test("expanded market includes newly verified remote job feeds", () => {
  const ids = new Set(DISCOVERY_PRESETS.map((preset) => preset.id));

  assert.ok(ids.has("remotefirstjobs-ai"), "expected RemoteFirstJobs AI preset");
  assert.ok(ids.has("remotefirstjobs-software-dev"), "expected RemoteFirstJobs software dev preset");
  assert.ok(ids.has("remotefirstjobs-golang"), "expected RemoteFirstJobs Golang preset");
  assert.ok(ids.has("realworkfromanywhere-backend"), "expected Real Work From Anywhere backend preset");
  assert.ok(ids.has("realworkfromanywhere-ai"), "expected Real Work From Anywhere AI preset");
  assert.ok(ids.has("jobicy-dev-fulltime"), "expected Jobicy dev full-time preset");
  assert.ok(ids.has("jobicy-data-science-fulltime"), "expected Jobicy data science full-time preset");
  assert.ok(ids.has("remoteok-engineering"), "expected Remote OK engineering preset");
  assert.ok(ids.has("smartremotejobs-devops"), "expected SmartRemoteJobs DevOps preset");
  assert.ok(ids.has("smartremotejobs-data-science"), "expected SmartRemoteJobs data science preset");
});

test("legacy preset groups are not used anymore", () => {
  for (const preset of DISCOVERY_PRESETS) {
    assert.notEqual(preset.group, "priority");
    assert.notEqual(preset.group, "general");
  }
});
