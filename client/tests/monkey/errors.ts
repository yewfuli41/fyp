// Wires up listeners that catch uncaught JS exceptions and console errors
// thrown by the page under test, and provides a helper that turns a captured
// error into a rich, reproducible test failure (screenshot + URL + action
// number + seed).

import type { Page, TestInfo } from "@playwright/test";

export interface CapturedError {
  kind: "pageerror" | "console";
  message: string;
  stack?: string;
}

/** Polled by the action loop so it can stop early once a fatal JS error has
 *  been captured, instead of grinding through the remaining iterations. */
export type ErrorTracker = {
  errors: CapturedError[];
  hasError(): boolean;
};

/**
 * Ignore-list for console messages that are noisy but not indicative of an
 * application bug. Extend this array if your app has known, harmless
 * `console.error` calls you don't want to fail the monkey run on.
 */
function isIgnorableConsoleMessage(text: string): boolean {
  const ignoredPatterns = [/ResizeObserver loop/i];
  return ignoredPatterns.some((pattern) => pattern.test(text));
}

/**
 * Attaches `pageerror` and console-error listeners to the page and returns a
 * tracker the caller can inspect. Errors are pushed in the order they occur,
 * so the first one captured is the one reported on failure.
 */
export function trackPageErrors(page: Page): ErrorTracker {
  const errors: CapturedError[] = [];

  page.on("pageerror", (error) => {
    errors.push({ kind: "pageerror", message: error.message, stack: error.stack });
  });

  page.on("console", (message) => {
    if (message.type() !== "error") return;
    const text = message.text();
    if (isIgnorableConsoleMessage(text)) return;
    errors.push({ kind: "console", message: text });
  });

  return {
    errors,
    hasError: () => errors.length > 0,
  };
}

/**
 * Captures everything needed to reproduce a failure — a screenshot, the
 * current URL, which action number it happened around, and the seed that
 * drove the run — attaches the screenshot to the HTML report, then throws so
 * the test fails with a descriptive message.
 */
export async function reportFatalError(
  page: Page,
  testInfo: TestInfo,
  tracker: ErrorTracker,
  seed: string,
  actionNumber: number
): Promise<never> {
  const screenshotPath = testInfo.outputPath(`monkey-failure-action-${actionNumber}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true }).catch(() => {
    // Best-effort — a detached/crashed page shouldn't stop us from reporting the error itself.
  });
  await testInfo
    .attach("failure-screenshot", { path: screenshotPath, contentType: "image/png" })
    .catch(() => {});

  const url = safePageUrl(page);
  const summary = tracker.errors.map((e, i) => `  [${i + 1}] (${e.kind}) ${e.message}`).join("\n");

  throw new Error(
    [
      `Monkey test caught ${tracker.errors.length} JS error(s).`,
      `Seed: ${seed}`,
      `Action number: ${actionNumber}`,
      `URL at failure: ${url}`,
      `Screenshot: ${screenshotPath}`,
      `Errors:`,
      summary,
    ].join("\n")
  );
}

function safePageUrl(page: Page): string {
  try {
    return page.url();
  } catch {
    return "<unavailable>";
  }
}
