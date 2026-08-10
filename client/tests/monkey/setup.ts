// Deterministic, non-random setup flow that gets a fresh browser session
// into "Business Mode": sign up a brand-new user, register a business
// profile for them, then switch the navbar's role switcher into business
// mode. Each step uses ordinary, targeted Playwright actions (not the random
// monkey actions) and throws with a clear message if it doesn't reach the
// expected state — a broken sign-up/registration flow is a real bug and
// should fail the test loudly right here, not leave the random loop quietly
// exploring the wrong (logged-out) pages.

import type { Page } from "@playwright/test";
import type { SeededRandom } from "./random";

export interface GeneratedAccount {
  username: string;
  email: string;
  password: string;
  contactNumber: string;
}

export interface GeneratedBusiness {
  businessName: string;
  address: string;
  contactNumber: string;
  email: string;
}

/**
 * Generates a valid-looking account. Derived from the seed's RNG so it's
 * traceable back to a run, but salted with the wall-clock time so re-running
 * the same seed doesn't collide with a previous run's now-taken email.
 */
export function generateAccount(rng: SeededRandom): GeneratedAccount {
  const suffix = `${Date.now()}_${rng.int(1000, 9999)}`;
  return {
    username: `monkey_${suffix}`.slice(0, 100),
    email: `monkey_${suffix}@example.com`,
    password: `Monkey!${rng.int(100_000, 999_999)}`,
    contactNumber: `01${rng.int(10_000_000, 99_999_999)}`,
  };
}

/** Generates a valid-looking business profile, same reasoning as {@link generateAccount}. */
export function generateBusiness(rng: SeededRandom): GeneratedBusiness {
  const suffix = `${Date.now()}_${rng.int(1000, 9999)}`;
  return {
    businessName: `Monkey Test Biz ${suffix}`,
    address: `${rng.int(1, 999)} Random Street`,
    contactNumber: `01${rng.int(10_000_000, 99_999_999)}`,
    email: `monkey_biz_${suffix}@example.com`,
  };
}

/**
 * Fills out and submits the sign-up form, and waits for the redirect to "/"
 * that follows a successful sign-up (the app logs the new user in
 * immediately — see SignUpPage.tsx) as proof it actually worked.
 */
export async function signUpNewUser(page: Page, account: GeneratedAccount): Promise<void> {
  await page.goto("/signup");

  // SignUpPage's fields are properly labelled (Form.Group controlId), so
  // accessible-name lookups work directly.
  await page.getByLabel("Username").fill(account.username);
  await page.getByLabel("Email").fill(account.email);
  await page.getByLabel("Contact number").fill(account.contactNumber);
  await page.getByLabel("Password", { exact: true }).fill(account.password);
  await page.getByLabel("Confirm password").fill(account.password);

  // The navbar also renders a "Sign up" link/button on every page (including
  // this one, until login succeeds), so scope to the form itself to avoid
  // ambiguity between the two.
  await page.locator(".auth-card").getByRole("button", { name: "Sign up" }).click();

  await page.waitForURL((url) => url.pathname === "/", { timeout: 10_000 });
}

/**
 * Fills out and submits the "Register Business Profile" form (leaving the
 * default working hours as-is — they're pre-filled with a valid entry) and
 * waits for the redirect to "/profile" that follows success.
 */
export async function registerBusiness(page: Page, business: GeneratedBusiness): Promise<void> {
  await page.goto("/register-business");

  // Unlike SignUpPage, these Form.Group elements don't set a controlId, so
  // there's no programmatic label→input association for getByLabel to use.
  // Fall back to "the input immediately following this label in the DOM".
  await fieldAfterLabel(page, "Business Name").fill(business.businessName);
  await fieldAfterLabel(page, "Address").fill(business.address);
  await fieldAfterLabel(page, "Business Contact Number").fill(business.contactNumber);
  await fieldAfterLabel(page, "Business Email").fill(business.email);

  await page.getByRole("button", { name: /register business/i }).click();

  await page.waitForURL((url) => url.pathname === "/profile", { timeout: 10_000 });
}

/**
 * Switches the navbar's role switcher into "Business Mode" — the same
 * control a real owner would use — and waits for the resulting navigation to
 * the Services page (see AppNavbar.tsx's switchMode).
 */
export async function switchToBusinessMode(page: Page): Promise<void> {
  // The role switcher is the only <select> on the page once logged in as an
  // owner outside of a form-heavy page like Register Business.
  const modeSwitcher = page.locator("select").first();
  await modeSwitcher.selectOption("business");
  await page.waitForURL((url) => url.pathname === "/services", { timeout: 10_000 });
}

/** Locates the `<input>`/`<textarea>` immediately following a text label in
 *  document order — a fallback for forms whose fields aren't wired up with a
 *  proper `for`/`id` association for `getByLabel` to use. */
function fieldAfterLabel(page: Page, labelText: string) {
  return page.getByText(labelText, { exact: true }).first().locator("xpath=following::input[1]");
}
