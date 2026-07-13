// Collects execution statistics for a monkey-test run — action counts,
// unique pages visited, most-frequently-clicked elements, timing, and JS
// error totals — then prints a console summary and/or serializes the whole
// thing to JSON for later analysis.
//
// Purely observational: nothing in this module influences which random
// action runs next, how long the run takes, or whether the test passes —
// it only records what already happened.

import * as fs from "fs";
import * as path from "path";
import type { ActionResult } from "./actions";
import type { ErrorTracker } from "./errors";

export interface MonkeyStats {
  seed: string;
  actionsPlanned: number;
  actionsExecuted: number;
  executionTimeMs: number;
  executionTimeFormatted: string;
  counts: {
    clicks: number;
    fills: number;
    scrolls: number;
    keypresses: number;
    skipped: number;
    failed: number;
  };
  uniquePagesVisited: string[];
  mostFrequentlyClickedElements: { label: string; count: number }[];
  jsErrors: {
    count: number;
    errors: { kind: string; message: string }[];
  };
  generatedAt: string;
}

/**
 * Accumulates stats while the monkey loop runs. Call the `record*` methods
 * as things happen, then `finalize()` once at the end to get a plain,
 * serializable snapshot.
 */
export class MonkeyStatsCollector {
  private readonly startedAt = Date.now();
  private actionsExecuted = 0;
  private readonly counts = { clicks: 0, fills: 0, scrolls: 0, keypresses: 0, skipped: 0, failed: 0 };
  private readonly pagesVisited = new Set<string>();
  private readonly clickTargets = new Map<string, number>();

  /** Feed in the result of one successful `runRandomAction` call. */
  recordAction(result: ActionResult): void {
    this.actionsExecuted++;

    if (!result.performed) {
      this.counts.skipped++;
      return;
    }

    switch (result.type) {
      case "click":
        this.counts.clicks++;
        if (result.target) {
          this.clickTargets.set(result.target, (this.clickTargets.get(result.target) ?? 0) + 1);
        }
        break;
      case "fill":
        this.counts.fills++;
        break;
      case "scroll":
        this.counts.scrolls++;
        break;
      case "keypress":
        this.counts.keypresses++;
        break;
    }
  }

  /** Call when an action attempt throws and is swallowed by the loop. */
  recordFailedAction(): void {
    this.actionsExecuted++;
    this.counts.failed++;
  }

  /** Call whenever the current URL is known (e.g. once per iteration), to
   *  build the set of unique pages visited during the run. */
  recordPageVisit(url: string | undefined | null): void {
    if (url) this.pagesVisited.add(url);
  }

  /** Produces the final stats snapshot. `errorTracker` supplies the JS error
   *  count/details; `plannedActions` is the configured loop size. */
  finalize(seed: string, plannedActions: number, errorTracker: ErrorTracker): MonkeyStats {
    const executionTimeMs = Date.now() - this.startedAt;
    return {
      seed,
      actionsPlanned: plannedActions,
      actionsExecuted: this.actionsExecuted,
      executionTimeMs,
      executionTimeFormatted: formatDuration(executionTimeMs),
      counts: { ...this.counts },
      uniquePagesVisited: [...this.pagesVisited],
      mostFrequentlyClickedElements: topEntries(this.clickTargets, 10),
      jsErrors: {
        count: errorTracker.errors.length,
        errors: errorTracker.errors.map((e) => ({ kind: e.kind, message: e.message })),
      },
      generatedAt: new Date().toISOString(),
    };
  }
}

/** Returns the top `limit` entries of a label→count map, highest count first. */
function topEntries(counts: Map<string, number>, limit: number): { label: string; count: number }[] {
  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, limit)
    .map(([label, count]) => ({ label, count }));
}

/** Formats a millisecond duration as e.g. "2m 3s" or "45s". */
function formatDuration(ms: number): string {
  const totalSeconds = Math.round(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`;
}

/** Prints a human-readable summary of the stats to the console. */
export function logStatsSummary(stats: MonkeyStats): void {
  console.log("\n[monkey] ── Execution summary ──────────────────────────");
  console.log(`[monkey] seed: ${stats.seed}`);
  console.log(`[monkey] actions executed: ${stats.actionsExecuted}/${stats.actionsPlanned}`);
  console.log(`[monkey] total execution time: ${stats.executionTimeFormatted} (${stats.executionTimeMs}ms)`);
  console.log(
    `[monkey] clicks: ${stats.counts.clicks}, fills: ${stats.counts.fills}, ` +
      `scrolls: ${stats.counts.scrolls}, keypresses: ${stats.counts.keypresses}, ` +
      `skipped: ${stats.counts.skipped}, failed: ${stats.counts.failed}`
  );
  console.log(
    `[monkey] unique pages visited (${stats.uniquePagesVisited.length}): ${stats.uniquePagesVisited.join(", ") || "(none)"}`
  );
  console.log("[monkey] most frequently clicked elements:");
  if (stats.mostFrequentlyClickedElements.length === 0) {
    console.log("[monkey]   (none)");
  } else {
    for (const { label, count } of stats.mostFrequentlyClickedElements) {
      console.log(`[monkey]   ${count}x  "${label}"`);
    }
  }
  console.log(`[monkey] JS errors: ${stats.jsErrors.count}`);
  console.log("[monkey] ──────────────────────────────────────────────────\n");
}

/** Writes the stats snapshot to a JSON file for later analysis / CI artifact
 *  upload, creating the parent directory if needed. */
export function saveStatsToFile(stats: MonkeyStats, filePath: string): void {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(stats, null, 2), "utf-8");
  console.log(`[monkey] report written to ${filePath}`);
}
