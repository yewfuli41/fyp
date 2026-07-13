import { test, expect } from "@playwright/test";
import { resolveSeed, SeededRandom } from "./monkey/random";
import { generateAccount, generateBusiness, signUpNewUser, registerBusiness, switchToBusinessMode } from "./monkey/setup";
import { runRandomAction } from "./monkey/actions";

test("smoke: avoidClickTexts keeps Log out from ever firing", async ({ page }) => {
  const seed = resolveSeed();
  const rng = new SeededRandom(seed);
  const account = generateAccount(rng);
  const business = generateBusiness(rng);

  await signUpNewUser(page, account);
  await registerBusiness(page, business);
  await switchToBusinessMode(page);

  // Force the profile dropdown open repeatedly so "Log out" is visible as
  // often as possible, then hammer random clicks with the exclusion active.
  for (let i = 0; i < 200; i++) {
    // Re-open the dropdown every few iterations since a click elsewhere may close it.
    if (i % 5 === 0) {
      await page.getByRole("button", { name: new RegExp(account.username, "i") }).click().catch(() => {});
    }
    const focusedBefore = await page
      .evaluate(() => {
        const el = document.activeElement;
        return el ? `${el.tagName}${el.getAttribute("role") ? `[role=${el.getAttribute("role")}]` : ""} "${(el.textContent || "").trim()}"` : null;
      })
      .catch(() => null);
    let result;
    try {
      result = await runRandomAction(page, rng, { avoidClickTexts: ["Log out"] });
    } catch (e) {
      result = { detail: `threw: ${(e as Error).message}` };
    }
    const token = await page.evaluate(() => localStorage.getItem("token")).catch(() => null);
    console.log(`[probe] #${i} focused-before=${focusedBefore} action=${result.detail} token=${token ? "present" : "NULL"}`);
    expect(token, `token should still be present after action #${i} (focused-before=${focusedBefore}, action=${result.detail})`).not.toBeNull();
  }
});
