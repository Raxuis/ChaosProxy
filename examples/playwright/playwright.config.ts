import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  workers: 1,
  use: {
    baseURL: process.env.APP_URL ?? "http://localhost:3000",
  },
});
