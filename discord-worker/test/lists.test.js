import assert from "node:assert/strict";
import test from "node:test";
import nacl from "tweetnacl";
import { DISCORD } from "../src/pricepulse.js";
import { helpMessage, renderDeals, renderFree, renderPopular } from "../src/lists.js";
import { consumeQueue, handleInteraction } from "../src/index.js";

const frontendURL = "https://pricepulse.test";
const deals = {
  card_rate: 1000,
  tax_rate: 0.21,
  featured: [
    { title: "Balatro", store_name: "Steam", price: 5, currency: "USD", discount_percent: 50, url: "https://steam.example/balatro", image_url: "https://cdn.example/b.jpg" },
    { title: "Regional", store_name: "Steam", price: 4500, currency: "ARS", discount_percent: 25, url: "https://steam.example/r" },
  ],
  free: [
    { title: "Free Game", store_name: "Epic Games Store", price: 0, currency: "USD", discount_percent: 100, url: "https://epic.example/free", expires_at: "2026-11-20T16:00:00Z" },
    { title: "Free Game", store_name: "Epic Games Store", price: 0, currency: "USD", discount_percent: 100, url: "https://epic.example/free" },
  ],
};

function signedRequest(payload, key) {
  const body = JSON.stringify(payload);
  const timestamp = "1700000000";
  const signature = nacl.sign.detached(new TextEncoder().encode(`${timestamp}${body}`), key.secretKey);
  return new Request("https://worker.test/interactions", { method: "POST", headers: { "x-signature-ed25519": Buffer.from(signature).toString("hex"), "x-signature-timestamp": timestamp }, body });
}

test("/ofertas lists paid deals converted to ARS with IVA", () => {
  const message = renderDeals(deals, frontendURL);
  const text = message.embeds[0].description;
  assert.match(text, /1\. Balatro\*\* — \*\*\$\s?5\.000\*\* \(-50%\)/);
  assert.match(text, /Regional\*\* — \*\*\$\s?4\.500\*\*/);
  const filled = renderDeals({ ...deals, featured: deals.featured.slice(0, 1), discounts: [deals.featured[0], { title: "Extra", store_name: "Epic", price: 2, currency: "USD", url: "https://epic.example/x" }] }, frontendURL).embeds[0].description;
  assert.equal(filled.match(/Balatro/g).length, 1);
  assert.match(filled, /2\. Extra/);
  assert.match(message.embeds[0].footer.text, /IVA 21%/);
  assert.equal(message.components[0].components[0].url, frontendURL);
});

test("/gratis removes duplicates and shows the expiry date", () => {
  const text = renderFree(deals, frontendURL).embeds[0].description;
  assert.equal(text.match(/Free Game/g).length, 1);
  assert.match(text, /hasta el 20\/11/);
  assert.match(renderFree({ free: [] }, frontendURL).embeds[0].description, /no hay juegos gratis/i);
});

test("/populares shows the Steam price and a cheaper store only when it differs", () => {
  const text = renderPopular({
    card_rate: 1000,
    popular: [
      { rank: 1, title: "Cheaper Elsewhere", steam_price: { store_name: "Steam", price: 20, currency: "USD" }, best_price: { store_name: "GOG", price: 10, currency: "USD" } },
      { rank: 2, title: "Steam Best", steam_price: { store_name: "Steam", price: 0, currency: "USD" }, best_price: { store_name: "Steam", price: 0, currency: "USD" } },
      { rank: 3, title: "Unreleased" },
      { rank: 4, title: "Tie", steam_price: { store_name: "Steam", price: 30, currency: "USD" }, best_price: { store_name: "Epic", price: 30, currency: "USD" } },
    ],
  }, frontendURL).embeds[0].description;
  assert.match(text, /#1 Cheaper Elsewhere\*\* — Steam \*\*\$\s?20\.000\*\* · más barato \*\*\$\s?10\.000\*\* en GOG/);
  assert.match(text, /#2 Steam Best\*\* — Steam \*\*Gratis\*\*$/m);
  assert.match(text, /#3 Unreleased\*\* — Sin precio publicado/);
  assert.match(text, /#4 Tie\*\* — Steam \*\*\$\s?30\.000\*\*$/m);
});

test("list commands defer publicly and /ayuda answers privately right away", async () => {
  const key = nacl.sign.keyPair();
  const jobs = [];
  const env = { BACKEND_URL: "https://api.test", FRONTEND_URL: frontendURL, DISCORD_APPLICATION_ID: "app-1", DISCORD_PUBLIC_KEY: Buffer.from(key.publicKey).toString("hex"), DISCORD_JOBS: { send: async (job) => jobs.push(job) } };
  const base = { type: DISCORD.APPLICATION_COMMAND, application_id: "app-1", token: "tok", member: { user: { id: "1" } } };

  const deferred = await handleInteraction(signedRequest({ ...base, data: { name: "gratis" } }, key), env);
  assert.deepEqual(await deferred.json(), { type: DISCORD.DEFERRED_CHANNEL_MESSAGE });
  assert.deepEqual(jobs, [{ kind: "gratis", application_id: "app-1", interaction_token: "tok" }]);

  const help = await (await handleInteraction(signedRequest({ ...base, data: { name: "ayuda" } }, key), env)).json();
  assert.equal(help.type, DISCORD.CHANNEL_MESSAGE);
  assert.equal(help.data.flags, DISCORD.EPHEMERAL);
  assert.match(help.data.embeds[0].description, /\/gratis/);
  assert.equal(jobs.length, 1);
});

test("queued list jobs read the matching endpoint and edit the original message", async () => {
  const env = { BACKEND_URL: "https://api.test", FRONTEND_URL: frontendURL };
  const requested = [];
  let edited;
  const fetchImpl = async (url, init = {}) => {
    if (String(url).startsWith("https://discord.com")) { edited = JSON.parse(init.body); return new Response("{}"); }
    requested.push(new URL(url).pathname);
    return new Response(JSON.stringify(String(url).includes("discover") ? { popular: [] } : deals));
  };
  const message = (kind) => ({ body: { kind, application_id: "app-1", interaction_token: "tok" }, ack() {}, retry() { throw new Error("should not retry"); } });
  await consumeQueue({ messages: [message("ofertas"), message("populares")] }, env, { fetchImpl });
  assert.deepEqual(requested, ["/api/deals", "/api/discover"]);
  assert.match(edited.embeds[0].title, /Populares/);
});

test("help lists every command", () => {
  const text = helpMessage(frontendURL).embeds[0].description;
  for (const command of ["/precio", "/ofertas", "/gratis", "/populares"]) assert.ok(text.includes(command), command);
});
