import { test as base, expect, type APIRequestContext, type APIResponse, type Page } from "@playwright/test";

export type ScenarioState = {
  name: string;
  match: string;
  enabled: boolean;
  on_exhausted: "passthrough" | "repeat" | "last";
  steps: string[];
  served: number;
  next_step: number | null;
  exhausted: boolean;
};

export type RuleStats = {
  matched: number;
  faulted: number;
  faults: Record<string, number>;
};

export type ChaosStats = {
  published: number;
  faulted: number;
  dropped: number;
  subscribers: number;
  history: number;
  statuses: Record<string, number>;
  rules: Record<string, RuleStats>;
};

export type ChaosProxy = {
  reset(): Promise<void>;
  stats(): Promise<ChaosStats>;
  scenario(name: string): Promise<ScenarioState>;
  resetScenario(name: string): Promise<void>;
  setRuleEnabled(name: string, enabled: boolean): Promise<void>;
  forceFaults(page: Page, directives: string): Promise<void>;
  clearForcedFaults(page: Page): Promise<void>;
};

const controlURL = process.env.CHAOSPROXY_CONTROL_URL ?? "http://127.0.0.1:7071";

export const test = base.extend<{ chaos: ChaosProxy }>({
  chaos: async ({ playwright }, use) => {
    const control = await playwright.request.newContext({ baseURL: controlURL });
    const chaos = createChaosProxy(control);
    await chaos.reset();
    await use(chaos);
    await control.dispose();
  },
});

export { expect };

export function createChaosProxy(control: APIRequestContext): ChaosProxy {
  return {
    async reset() {
      await ensureOk(await control.post("/api/reset"), "reset");
    },
    async stats() {
      const response = await control.get("/api/stats");
      await ensureOk(response, "stats");
      return (await response.json()) as ChaosStats;
    },
    async scenario(name) {
      const response = await control.get("/api/scenarios");
      await ensureOk(response, "scenario lookup");
      const scenario = ((await response.json()) as ScenarioState[]).find((candidate) => candidate.name === name);
      if (!scenario) {
        throw new Error(`chaosproxy has no scenario named ${name}`);
      }
      return scenario;
    },
    async resetScenario(name) {
      await ensureOk(await control.post(`/api/scenarios/${encodeURIComponent(name)}/reset`), `reset of scenario ${name}`);
    },
    async setRuleEnabled(name, enabled) {
      await ensureOk(await control.put(`/api/rules/${encodeURIComponent(name)}`, { data: { enabled } }), `toggle of rule ${name}`);
    },
    async forceFaults(page, directives) {
      await page.setExtraHTTPHeaders({ "X-Chaos": directives });
    },
    async clearForcedFaults(page) {
      await page.setExtraHTTPHeaders({});
    },
  };
}

async function ensureOk(response: APIResponse, action: string): Promise<void> {
  if (!response.ok()) {
    throw new Error(`chaosproxy ${action} failed with ${response.status()}: ${await response.text()}`);
  }
}
