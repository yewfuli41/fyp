// The catalogue of "random actions" the monkey can perform, plus the
// registry that picks one at random each iteration.
//
// To extend the monkey with a new kind of action: write an async function
// matching the `MonkeyAction` signature and add it to the `ACTIONS` array.
// Nothing else needs to change — the loop in monkey.spec.ts just picks
// uniformly from whatever is in that list.

import type { Locator, Page } from "@playwright/test";
import type { SeededRandom } from "./random";

/** Selectors considered "clickable" for the purposes of random clicking. */
const CLICKABLE_SELECTOR = [
  "button",
  "a[href]",
  '[role="button"]',
  '[role="link"]',
  '[role="menuitem"]',
  '[role="tab"]',
  'input[type="submit"]',
  'input[type="button"]',
  'input[type="checkbox"]',
  'input[type="radio"]',
].join(", ");

/** Selectors considered "fillable" for the purposes of random text entry. */
const FILLABLE_SELECTOR = [
  'input:not([type="hidden"]):not([type="checkbox"]):not([type="radio"])' +
    ':not([type="submit"]):not([type="button"]):not([type="file"]):not([type="range"]):not([type="color"])',
  "textarea",
].join(", ");

const RANDOM_KEYS = ["Tab", "Escape", "Enter", "ArrowUp", "ArrowDown"] as const;

/** The category an action falls into, used for statistics tallying. */
export type ActionType = "click" | "fill" | "scroll" | "keypress";

/**
 * Outcome of a single action attempt. `detail` is the same human-readable
 * string the loop has always logged; `performed` and `target` are extra
 * metadata (added purely for statistics) that don't affect what the action
 * actually did.
 */
export interface ActionResult {
  type: ActionType;
  /** False when the action found nothing to act on and was skipped (e.g. no
   *  visible clickable/fillable element) — doesn't count as a real click/fill. */
  performed: boolean;
  /** Human-readable description, used for console logging. */
  detail: string;
  /** For clicks: a label identifying which element was clicked, used to
   *  tally "most frequently clicked elements". */
  target?: string;
}

/**
 * Options that tune how an action behaves, without changing what kind of
 * action it is. Every field is optional so existing callers (and other
 * `MonkeyAction`s that don't care about clicking) are unaffected.
 */
export interface MonkeyActionOptions {
  /** Click candidates whose visible text matches one of these (case-insensitive,
   *  trimmed) are skipped rather than clicked — e.g. so a monkey run driving
   *  an authenticated session doesn't randomly click "Log out" and spend its
   *  remaining iterations logged out. */
  avoidClickTexts?: string[];
}

/**
 * Signature every monkey action must implement: given the page and the
 * seeded RNG (so it can make its own randomized sub-decisions), perform one
 * action and return an `ActionResult` describing what happened.
 * Should throw on failure — the caller is responsible for catching it.
 */
export type MonkeyAction = (page: Page, rng: SeededRandom, options?: MonkeyActionOptions) => Promise<ActionResult>;

/**
 * Picks one random, visible & enabled element matching `selector` — skipping
 * any whose text matches `avoidTexts` — or `null` if none are currently
 * available/eligible. Re-queries the DOM fresh on every call (via a locator,
 * not a cached handle) so it never hands back a reference to something
 * that's since been detached from the page.
 */
async function pickRandomVisibleElement(
  page: Page,
  rng: SeededRandom,
  selector: string,
  avoidTexts: string[] = []
): Promise<Locator | null> {
  const candidates = page.locator(selector);
  const count = await candidates.count();
  if (count === 0) return null;

  const normalizedAvoidTexts = avoidTexts.map((t) => t.trim().toLowerCase());

  // Probe a handful of random indices rather than checking every element —
  // cheaper, and "some" random coverage per iteration is the goal here, not
  // exhaustiveness (we run 1000 iterations, so coverage compounds over time).
  const attempts = Math.min(count, 8);
  for (let i = 0; i < attempts; i++) {
    const index = rng.int(0, count - 1);
    const el = candidates.nth(index);
    try {
      if (!(await el.isVisible()) || !(await el.isEnabled())) continue;
      if (normalizedAvoidTexts.length > 0) {
        const text = ((await el.textContent().catch(() => "")) ?? "").trim().toLowerCase();
        if (normalizedAvoidTexts.includes(text)) continue;
      }
      return el;
    } catch {
      // Element vanished between count() and now (e.g. a re-render) — try another index.
    }
  }
  return null;
}

/** Clicks a random visible, enabled clickable element (excluding any listed
 *  in `options.avoidClickTexts`). */
const clickRandomElement: MonkeyAction = async (page, rng, options) => {
  const el = await pickRandomVisibleElement(page, rng, CLICKABLE_SELECTOR, options?.avoidClickTexts);
  if (!el) return { type: "click", performed: false, detail: "click: no clickable element available, skipped" };
  const label = (await el.textContent().catch(() => null))?.trim().slice(0, 40) || "<no text>";
  await el.click({ timeout: 3000 });
  return { type: "click", performed: true, detail: `click: "${label}"`, target: label };
};

/** Fills a random visible, enabled input/textarea with a random string. */
const fillRandomField: MonkeyAction = async (page, rng) => {
  const el = await pickRandomVisibleElement(page, rng, FILLABLE_SELECTOR);
  if (!el) return { type: "fill", performed: false, detail: "fill: no fillable field available, skipped" };
  const value = rng.string(3, 16);
  await el.fill(value, { timeout: 3000 });
  return { type: "fill", performed: true, detail: `fill: "${value}"` };
};

/** Scrolls the page by a random amount, in a random direction. */
const randomScroll: MonkeyAction = async (page, rng) => {
  const dx = rng.int(-300, 300);
  const dy = rng.int(-500, 500);
  await page.mouse.wheel(dx, dy);
  return { type: "scroll", performed: true, detail: `scroll: (${dx}, ${dy})` };
};

/** Returns the trimmed, lowercased text of the currently focused element, or
 *  `null` if nothing meaningful is focused. Used to avoid pressing Enter on
 *  an avoided element that Tab happened to focus. */
async function focusedElementText(page: Page): Promise<string | null> {
  try {
    const text = await page.evaluate(() => {
      const el = document.activeElement;
      if (!el || el === document.body) return null;
      return el.textContent;
    });
    return text?.trim().toLowerCase() || null;
  } catch {
    return null;
  }
}

/**
 * Presses a random "navigation-ish" key. Tab can move focus onto any
 * focusable element — including one listed in `options.avoidClickTexts` —
 * and Enter would then activate it exactly like a click would, so Enter is
 * skipped whenever the currently focused element is on the avoid-list.
 */
const randomKeyPress: MonkeyAction = async (page, rng, options) => {
  const key = rng.pick(RANDOM_KEYS);

  if (key === "Enter" && options?.avoidClickTexts?.length) {
    const focusedText = await focusedElementText(page);
    const avoided = focusedText && options.avoidClickTexts.some((t) => t.trim().toLowerCase() === focusedText);
    if (avoided) {
      return {
        type: "keypress",
        performed: false,
        detail: `keypress: Enter skipped (focused element "${focusedText}" is avoided)`,
      };
    }
  }

  await page.keyboard.press(key);
  return { type: "keypress", performed: true, detail: `keypress: ${key}` };
};

/**
 * The full set of actions the monkey can choose from. Every entry is picked
 * with equal probability by default — add more entries here to extend the
 * monkey's repertoire.
 */
export const ACTIONS: MonkeyAction[] = [clickRandomElement, fillRandomField, randomScroll, randomKeyPress];

/** Picks and runs one random action, returning its `ActionResult` (used both
 *  for the console log line and for statistics collection). */
export async function runRandomAction(
  page: Page,
  rng: SeededRandom,
  options?: MonkeyActionOptions
): Promise<ActionResult> {
  const action = rng.pick(ACTIONS);
  return action(page, rng, options);
}
