const apiOrigin = process.env.CHAOSPROXY_URL ?? process.env.API_ORIGIN ?? "http://localhost:9000";

export type Account = {
  user: { name: string; email: string | null };
};

export async function getAccount(): Promise<Account> {
  const response = await fetch(`${apiOrigin}/api/account`, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`account request failed with ${response.status}`);
  }
  return (await response.json()) as Account;
}
