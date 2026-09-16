import { expect, test } from "./chaos";

test("payment succeeds after the API fails twice", async ({ page, chaos }) => {
  await page.goto("/checkout");
  await page.getByRole("button", { name: "Pay now" }).click();

  await expect(page.getByText("Payment confirmed")).toBeVisible();
  const payments = await chaos.scenario("payment-recovers");
  expect(payments.served).toBe(3);
  expect(payments.exhausted).toBe(true);
});

test("orders page explains an outage", async ({ page, chaos }) => {
  await chaos.forceFaults(page, "status=503");
  await page.goto("/orders");

  await expect(page.getByRole("alert")).toContainText("temporarily unavailable");
});
