// Random "monkey" UI testing: hammers the app with 1000 random actions
// (clicks, fills, scrolls, key presses) looking for uncaught JS errors.
//
// This variant explores the app as an anonymous visitor, starting from "/".
// For a variant that signs up, registers a business, and explores the
// authenticated "Business Mode" surface instead, see monkey-business.spec.ts.
//
// Reproducibility: the whole run is driven by a single seed, printed at the
// start. Rerun a failing session with the exact same sequence of actions via:
//
//   MONKEY_SEED=<seed-from-log> npx playwright test tests/monkey.spec.ts
//
// If MONKEY_SEED isn't set, a fresh seed is generated and printed instead.

import * as path from "path";
import { test, expect } from "@playwright/test";
import { resolveSeed, SeededRandom } from "./monkey/random";
import { trackPageErrors, reportFatalError } from "./monkey/errors";
import { MonkeyStatsCollector, logStatsSummary, saveStatsToFile } from "./monkey/stats";
import { runMonkeyLoop } from "./monkey/loop";

const TOTAL_ACTIONS = 1000;
const MIN_DELAY_MS = 100;
const MAX_DELAY_MS = 500;
const REPORT_PATH = path.join(process.cwd(), "monkey-report.json");

test.describe("Monkey testing", () => {
  test("performs 1000 random UI actions without triggering a JS error", async ({ page }, testInfo) => {
    // Worst case is TOTAL_ACTIONS back-to-back MAX_DELAY_MS waits, plus
    // per-action overhead — give the run generous headroom over that.
    test.setTimeout(TOTAL_ACTIONS * (MAX_DELAY_MS + 200) + 60_000);

    const seed = resolveSeed();
    console.log(`[monkey] seed = ${seed}  (rerun with: MONKEY_SEED=${seed})`);
    const rng = new SeededRandom(seed);

    const errorTracker = trackPageErrors(page);
    const stats = new MonkeyStatsCollector();

    // Dialogs (alert/confirm/prompt) block the page until dismissed — auto-dismiss
    // so the monkey never gets stuck waiting on one.
    page.on("dialog", (dialog) => {
      dialog.dismiss().catch(() => {
        // Dialog may already be gone (e.g. page navigated away) — nothing to do.
      });
    });

    await page.goto("/");

    const actionNumber = await runMonkeyLoop({
      page,
      rng,
      totalActions: TOTAL_ACTIONS,
      minDelayMs: MIN_DELAY_MS,
      maxDelayMs: MAX_DELAY_MS,
      errorTracker,
      stats,
    });

    // Stats are logged and saved regardless of outcome — they're at least as
    // useful for diagnosing a failed run as for a clean one.
    const finalStats = stats.finalize(seed, TOTAL_ACTIONS, errorTracker);
    logStatsSummary(finalStats);
    saveStatsToFile(finalStats, REPORT_PATH);
    await testInfo.attach("monkey-report", { path: REPORT_PATH, contentType: "application/json" }).catch(() => {});

    if (errorTracker.hasError()) {
      await reportFatalError(page, testInfo, errorTracker, seed, actionNumber);
    }

    expect(errorTracker.errors, `Seed ${seed} produced JS errors`).toHaveLength(0);
  });
});
