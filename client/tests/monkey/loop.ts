// The core "keep clicking/filling/scrolling/pressing keys at random" loop,
// shared by every monkey-test variant (anonymous browsing, business-owner
// flows, ...). Each variant is responsible for getting the page into
// whatever starting state it wants (logged out, logged in as an owner,
// etc.) and wiring up its own error tracker/stats collector — this module
// just runs the loop against them.

import type { Page } from "@playwright/test";
import type { SeededRandom } from "./random";
import type { ErrorTracker } from "./errors";
import type { MonkeyStatsCollector } from "./stats";
import { runRandomAction } from "./actions";
import type { MonkeyActionOptions } from "./actions";

export interface MonkeyLoopOptions {
  page: Page;
  rng: SeededRandom;
  totalActions: number;
  minDelayMs: number;
  maxDelayMs: number;
  errorTracker: ErrorTracker;
  stats: MonkeyStatsCollector;
  /** Print a progress line every N actions (default 50). */
  logEvery?: number;
  /** Forwarded to every `runRandomAction` call — e.g. `{ avoidClickTexts: ["Log out"] }`
   *  to keep an authenticated run from ending its own session early. */
  actionOptions?: MonkeyActionOptions;
}

/**
 * Runs the random-action loop until `totalActions` iterations complete or a
 * fatal JS error is captured (whichever happens first). Individual action
 * failures are logged and swallowed — only errors surfaced through
 * `errorTracker` (pageerror / console.error) stop the loop early.
 *
 * Returns the number of iterations actually run.
 */
export async function runMonkeyLoop(options: MonkeyLoopOptions): Promise<number> {
  const { page, rng, totalActions, minDelayMs, maxDelayMs, errorTracker, stats, logEvery = 50, actionOptions } = options;

  stats.recordPageVisit(currentUrlSafe(page));

  let actionNumber = 0;
  for (; actionNumber < totalActions; actionNumber++) {
    // A fatal JS error was already captured by the listeners in errors.ts —
    // stop grinding through the remaining iterations.
    if (errorTracker.hasError()) break;

    try {
      const result = await runRandomAction(page, rng, actionOptions);
      stats.recordAction(result);
      if (actionNumber % logEvery === 0) {
        console.log(`[monkey] #${actionNumber}: ${result.detail}`);
      }
    } catch (error) {
      // Individual action failures — detached elements, elements that became
      // unavailable mid-action, navigation racing the action, etc. — are
      // expected noise for a monkey test. Log and keep going; only actual
      // page JS errors (captured via errorTracker above) are fatal.
      stats.recordFailedAction();
      console.log(`[monkey] #${actionNumber} action failed (ignored): ${(error as Error).message}`);
    }

    // Track unique pages regardless of which branch above ran — a click can
    // trigger a client-side route change even when the action itself throws.
    stats.recordPageVisit(currentUrlSafe(page));

    const delay = minDelayMs + rng.int(0, maxDelayMs - minDelayMs);
    await page.waitForTimeout(delay);
  }

  console.log(`[monkey] finished after ${actionNumber} action(s), ${errorTracker.errors.length} JS error(s) captured`);
  return actionNumber;
}

/** Best-effort current URL — the page can be mid-navigation or closed when
 *  this is called, and stats collection should never be the reason an
 *  iteration throws. */
function currentUrlSafe(page: Page): string | null {
  try {
    return page.url();
  } catch {
    return null;
  }
}
