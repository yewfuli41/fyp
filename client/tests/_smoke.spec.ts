import { test, expect } from "@playwright/test";
import { resolveSeed, SeededRandom } from "./monkey/random";
import { generateAccount, generateBusiness, signUpNewUser, registerBusiness, switchToBusinessMode } from "./monkey/setup";
import { runRandomAction } from "./monkey/actions";

test("smoke: avoidClickTexts keeps Log out from ever firing", async ({ page }) => {
  test.setTimeout(180_000);
  const seed = resolveSeed();
  const rng = new SeededRandom(seed);
  const account = generateAccount(rng);
  const business = generateBusiness(rng);

  await signUpNewUser(page, account);
  await registerBusiness(page, business);
  await switchToBusinessMode(page);

  for (let i = 0; i < 200; i++) {
    if (i % 5 === 0) {
      await page.getByRole("button", { name: new RegExp(account.username, "i") }).click().catch(() => {});
    }
    let detail = "";
    try {
      const result = await runRandomAction(page, rng, { avoidClickTexts: ["Log out"] });
      detail = result.detail;
    } catch (e) {
      detail = `threw: ${(e as Error).message}`;
    }
    const token = await page.evaluate(() => localStorage.getItem("token")).catch(() => null);
    expect(token, `token should still be present after action #${i} (${detail})`).not.toBeNull();
  }
});
