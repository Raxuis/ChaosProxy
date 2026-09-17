import { chromium } from "playwright-core";

const out = process.argv[2];
const app = "http://localhost:3001/";
const dashboard = "http://localhost:7071/";
const frames = [];

const browser = await chromium.launch({ channel: "chrome", headless: true });
const context = await browser.newContext({ viewport: { width: 1280, height: 720 }, deviceScaleFactor: 2 });
const page = await context.newPage();
const errors = [];
page.on("pageerror", (error) => errors.push(error.message));

async function shot(name) {
  const file = `${out}/${String(frames.length).padStart(2, "0")}-${name}.png`;
  await page.screenshot({ path: file });
  frames.push(file);
}

async function setRule(name, enabled) {
  await page.request.put(`${dashboard}api/rules/${name}`, { data: { enabled } });
}

await page.request.post(`${dashboard}api/reset`);
await setRule("one-null-article", false);

await page.goto(app);
await page.locator(".article-preview h1").first().waitFor();
await shot("app-ok");

await page.goto(dashboard);
await page.locator("#feed tr").nth(1).waitFor();
await page.waitForTimeout(400);
await shot("dashboard-ok");

const toggle = page.getByRole("switch", { name: "Inject faults for one-null-article" });
await toggle.click();
await page.waitForTimeout(500);
await shot("toggle-on");

await page.goto(app);
await page.waitForTimeout(2000);
await shot("app-stuck");

await page.goto(dashboard);
await page.locator("#feed tr").nth(3).waitFor();
await page.waitForTimeout(400);
await shot("dashboard-mutated");

await setRule("one-null-article", false);
await page.goto(app);
await page.locator(".article-preview h1").first().waitFor();
await shot("app-recovered");

await browser.close();
console.log(JSON.stringify({ frames, errors }, null, 2));
