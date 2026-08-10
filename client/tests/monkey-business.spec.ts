// Monkey testing scoped to the business-owner ("Business Mode") experience.
//
// monkey.spec.ts explores the app as an anonymous visitor. This variant
// instead drives a real sign-up → register-business → switch-to-business-mode
// flow with ordinary, targeted Playwright actions first, and only starts
// throwing random actions at the page once that setup succeeds — so the
// monkey loop spends its time on the authenticated, owner-only surface
// (Dashboard, Services, Staff, Service Slots, Edit Business, ...) instead of
// just the public pages.
//
// Reproducibility: the seed drives both the generated account/business data
// (see monkey/setup.ts) and the random action sequence that follows, so
// MONKEY_SEED=<seed> replays the same run. Note the generated email/username
// are salted with the wall-clock time so a rerun doesn't fail sign-up on a
// "this email is already taken" collision with the original run's account.

import * as path from "path";
import { test, expect } from "@playwright/test";
import { resolveSeed, SeededRandom } from "./monkey/random";
import { trackPageErrors, reportFatalError } from "./monkey/errors";
import { MonkeyStatsCollector, logStatsSummary, saveStatsToFile } from "./monkey/stats";
import { runMonkeyLoop } from "./monkey/loop";
import {
  generateAccount,
  generateBusiness,
  signUpNewUser,
  registerBusiness,
  switchToBusinessMode,
} from "./monkey/setup";

const TOTAL_ACTIONS = 1000;
const MIN_DELAY_MS = 100;
const MAX_DELAY_MS = 500;
const REPORT_PATH = path.join(process.cwd(), "monkey-business-report.json");

test.describe("Monkey testing — business owner mode", () => {
  test("signs up, registers a business, switches to business mode, then performs 1000 random UI actions", async ({
    page,
  }, testInfo) => {
    // Worst case is TOTAL_ACTIONS back-to-back MAX_DELAY_MS waits, plus
    // per-action overhead and the setup flow — give the run generous headroom.
    test.setTimeout(TOTAL_ACTIONS * (MAX_DELAY_MS + 200) + 90_000);

    const seed = resolveSeed();
    console.log(`[monkey] seed = ${seed}  (rerun with: MONKEY_SEED=${seed})`);
    const rng = new SeededRandom(seed);

    const errorTracker = trackPageErrors(page);
    const stats = new MonkeyStatsCollector();

    // Dialogs (alert/confirm/prompt) block the page until dismissed — auto-dismiss
    // so the monkey never gets stuck waiting on one, during setup or the loop.
    page.on("dialog", (dialog) => {
      dialog.dismiss().catch(() => {
        // Dialog may already be gone (e.g. page navigated away) — nothing to do.
      });
    });

    // --- Setup: get into Business Mode before any random action runs. ---
    // Deliberately NOT wrapped in a try/catch that swallows failures — if
    // sign-up/registration is broken, that's a real bug and the test should
    // fail clearly here rather than the monkey loop quietly exploring the
    // wrong (logged-out) pages for the next several minutes.
    const account = generateAccount(rng);
    const business = generateBusiness(rng);

    console.log(`[monkey] setup: signing up as ${account.email}`);
    await signUpNewUser(page, account);

    console.log(`[monkey] setup: registering business "${business.businessName}"`);
    await registerBusiness(page, business);

    console.log(`[monkey] setup: switching to business mode`);
    await switchToBusinessMode(page);

    console.log(`[monkey] setup complete — starting random actions from ${page.url()}`);

    // --- Random phase, identical machinery to monkey.spec.ts, except random
    // clicks must never hit "Log out" — that would end the authenticated
    // session and leave the remaining iterations exploring the logged-out
    // app instead of Business Mode. ---
    const actionNumber = await runMonkeyLoop({
      page,
      rng,
      totalActions: TOTAL_ACTIONS,
      minDelayMs: MIN_DELAY_MS,
      maxDelayMs: MAX_DELAY_MS,
      errorTracker,
      stats,
      actionOptions: { avoidClickTexts: ["Log out"] },
    });

    const finalStats = stats.finalize(seed, TOTAL_ACTIONS, errorTracker);
    logStatsSummary(finalStats);
    saveStatsToFile(finalStats, REPORT_PATH);
    await testInfo.attach("monkey-business-report", { path: REPORT_PATH, contentType: "application/json" }).catch(() => {});

    if (errorTracker.hasError()) {
      await reportFatalError(page, testInfo, errorTracker, seed, actionNumber);
    }

    expect(errorTracker.errors, `Seed ${seed} produced JS errors`).toHaveLength(0);
  });
});
