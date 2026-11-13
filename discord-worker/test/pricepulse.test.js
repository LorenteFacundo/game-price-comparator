import assert from "node:assert/strict";
import test from "node:test";
import nacl from "tweetnacl";
import {
  DISCORD,
  makeCursor,
  parseCursor,
  rankMatches,
  renderResult,
  selectPrices,
  validateQuery,
  verifyDiscordSignature,
} from "../src/pricepulse.js";
import { consumeQueue, handleInteraction } from "../src/index.js";

const env = {
  BACKEND_URL: "https://api.pricepulse.test",
  FRONTEND_URL: "https://pricepulse.test",
  DISCORD_APPLICATION_ID: "app-1",
  DISCORD_PUBLIC_KEY: "",
  DISCORD_JOBS: { send: async () => {} },
};

const responseData = {
  card_rate: 1500,
  tax_rate: 0.21,
  rates_updated_at: "2026-09-11T12:00:00Z",
  results: [{
    title: "Hades",
    image_url: "https://cdn.example/hades.jpg",
    prices: [
      { store_name: "Steam", price: 10, currency: "USD", discount_percent: 50, url: "https://steam.example/hades" },
      { store_name: "Epic Games Store", price: 8000, currency: "ARS", discount_percent: 10, url: "https://epic.example/hades" },
      { store_name: "Microsoft Store", price: 0, currency: "ARS", discount_percent: 100, url: "https://microsoft.example/hades" },
      { store_name: "Fanatical", price: 5, currency: "USD", discount_percent: 20, url: "https://fanatical.example/hades" },
    ],
  }, { title: "Hades II", image_url: "", prices: [] }],
};

function interactionRequest(payload, signingKey) {
  const body = JSON.stringify(payload);
  const timestamp = "1700000000";
  const signature = nacl.sign.detached(new TextEncoder().encode(`${timestamp}${body}`), signingKey.secretKey);
  return new Request("https://worker.test/interactions", {
    method: "POST",
    headers: {
      "x-signature-ed25519": Buffer.from(signature).toString("hex"),
      "x-signature-timestamp": timestamp,
    },
    body,
  });
}

test("verifies a valid Discord Ed25519 signature and rejects invalid signatures", () => {
  const key = nacl.sign.keyPair();
  const timestamp = "1700000000";
  const body = '{"type":1}';
  const signature = nacl.sign.detached(new TextEncoder().encode(`${timestamp}${body}`), key.secretKey);
  assert.equal(verifyDiscordSignature(Buffer.from(key.publicKey).toString("hex"), Buffer.from(signature).toString("hex"), timestamp, body), true);
  assert.equal(verifyDiscordSignature(Buffer.from(key.publicKey).toString("hex"), "00".repeat(64), timestamp, body), false);
});

test("answers PING and rejects an invalid interaction signature", async () => {
  const key = nacl.sign.keyPair();
  const configured = { ...env, DISCORD_PUBLIC_KEY: Buffer.from(key.publicKey).toString("hex") };
  const pong = await handleInteraction(interactionRequest({ type: DISCORD.PING }, key), configured);
  assert.deepEqual(await pong.json(), { type: DISCORD.PONG });

  const invalid = await handleInteraction(new Request("https://worker.test/interactions", { method: "POST", body: "{}" }), configured);
  assert.equal(invalid.status, 401);
});

test("/precio defers publicly and queues exactly one request", async () => {
  const key = nacl.sign.keyPair();
  const jobs = [];
  const configured = { ...env, DISCORD_PUBLIC_KEY: Buffer.from(key.publicKey).toString("hex"), DISCORD_JOBS: { send: async (job) => jobs.push(job) } };
  const request = interactionRequest({
    type: DISCORD.APPLICATION_COMMAND,
    application_id: "app-1",
    token: "token",
    member: { user: { id: "123" } },
    data: { name: "precio", options: [{ name: "juego", value: "Hades" }] },
  }, key);
  const result = await handleInteraction(request, configured);
  assert.deepEqual(await result.json(), { type: DISCORD.DEFERRED_CHANNEL_MESSAGE });
  assert.deepEqual(jobs, [{ application_id: "app-1", interaction_token: "token", query: "Hades", result_index: 0, requester_id: "123", update_type: "command" }]);
});

test("validates query byte length and protects navigation from other users", async () => {
  assert.match(validateQuery("á".repeat(26)), /50 bytes/);
  const key = nacl.sign.keyPair();
  const cursor = makeCursor("n", 1, "123", "Hades");
  const configured = { ...env, DISCORD_PUBLIC_KEY: Buffer.from(key.publicKey).toString("hex") };
  const request = interactionRequest({ type: DISCORD.MESSAGE_COMPONENT, application_id: "app-1", token: "token", member: { user: { id: "456" } }, data: { custom_id: cursor } }, key);
  const result = await handleInteraction(request, configured);
  const payload = await result.json();
  assert.equal(payload.data.flags, DISCORD.EPHEMERAL);
  assert.match(payload.data.content, /Sólo quien inició/);
});

test("cursor retains query, index and owner within Discord's custom_id limit", () => {
  const cursor = makeCursor("n", 12, "1234567890123456789", "a".repeat(50));
  assert.ok(cursor.length <= 100);
  assert.deepEqual(parseCursor(cursor), { direction: "n", index: 12, requesterId: "1234567890123456789", query: "a".repeat(50) });
  assert.equal(parseCursor("p:n:0:1:invalid*"), null);
});

test("ranks exact match first, keeps free games and selects top three plus priority stores", () => {
  assert.deepEqual(rankMatches([{ title: "Hades II" }, { title: "Hades" }], "Hades").map((item) => item.title), ["Hades", "Hades II"]);
  const selection = selectPrices(responseData.results[0].prices, responseData.card_rate);
  assert.equal(selection[0].store_name, "Microsoft Store");
  assert.equal(selection[0].finalARS, 0);
  assert.equal(selection.find((price) => price.store_name === "Epic Games Store").finalARS, 8000);
  assert.equal(selection.find((price) => price.store_name === "Steam").finalARS, 15000);
  assert.ok(selection.some((price) => price.store_name === "Steam"));
  assert.ok(selection.some((price) => price.store_name === "Epic Games Store"));
  assert.ok(selection.length <= 6);
});

test("renders compact ARS + IVA result with navigation boundaries", () => {
  const payload = renderResult({ response: responseData, game: responseData.results[0], query: "Hades", resultIndex: 0, requesterId: "123", frontendURL: env.FRONTEND_URL });
  assert.match(payload.embeds[0].fields[0].value, /Gratis/);
  assert.match(payload.embeds[0].footer.text, /IVA 21%/);
  assert.equal(payload.components[0].components[0].disabled, true);
  assert.equal(payload.components[0].components[1].disabled, false);
});

test("queue edits the original answer once and retries failed Discord edits", async () => {
  let patchCalls = 0;
  const fetchImpl = async (url, init = {}) => {
    if (String(url).includes("/api/search")) return new Response(JSON.stringify(responseData));
    patchCalls += 1;
    assert.equal(init.method, "PATCH");
    return new Response("{}", { status: 200 });
  };
  const message = { body: { application_id: "app-1", interaction_token: "token", query: "Hades", result_index: 0, requester_id: "123" }, acked: 0, retried: 0, ack() { this.acked += 1; }, retry() { this.retried += 1; } };
  await consumeQueue({ messages: [message] }, env, { fetchImpl });
  assert.equal(message.acked, 1);
  assert.equal(message.retried, 0);
  assert.equal(patchCalls, 1);

  const retryMessage = { ...message, acked: 0, retried: 0, ack() { this.acked += 1; }, retry() { this.retried += 1; } };
  await consumeQueue({ messages: [retryMessage] }, env, { fetchImpl: async () => { throw new Error("network"); } });
  assert.equal(retryMessage.acked, 0);
  assert.equal(retryMessage.retried, 1);
});

test("queue returns friendly no-results and provider-failure messages without exposing internals", async () => {
  const patches = [];
  const message = { body: { application_id: "app-1", interaction_token: "token", query: "Unknown", result_index: 0, requester_id: "123" }, acked: 0, retried: 0, ack() { this.acked += 1; }, retry() { this.retried += 1; } };
  const fetchImpl = async (url, init = {}) => {
    if (String(url).includes("/api/search")) return new Response(JSON.stringify({ results: [] }));
    patches.push(JSON.parse(init.body));
    return new Response("{}", { status: 200 });
  };
  await consumeQueue({ messages: [message] }, env, { fetchImpl });
  assert.match(patches[0].content, /No encontré coincidencias/);
  assert.equal(message.acked, 1);

  const outage = { ...message, acked: 0, retried: 0, ack() { this.acked += 1; }, retry() { this.retried += 1; } };
  const outagePatches = [];
  await consumeQueue({ messages: [outage] }, env, { fetchImpl: async (url, init = {}) => {
    if (String(url).includes("/api/search")) return new Response("unavailable", { status: 502 });
    outagePatches.push(JSON.parse(init.body));
    return new Response("{}", { status: 200 });
  } });
  assert.match(outagePatches[0].content, /No pude consultar los precios/);
  assert.equal(outage.acked, 1);
});

test("edits the original Discord response with a bot User-Agent", async () => {
  const env = { BACKEND_URL: "https://api.example", FRONTEND_URL: "https://web.example" };
  let discordHeaders;
  let discordURL;
  const fetchImpl = async (url, init = {}) => {
    if (String(url).startsWith("https://discord.com")) {
      discordHeaders = init.headers;
      discordURL = String(url);
      return new Response("{}", { status: 200 });
    }
    return new Response(JSON.stringify({ results: [] }), { status: 200 });
  };
  const message = { body: { application_id: "app-1", interaction_token: "token", query: "Hades", result_index: 0, requester_id: "123" }, ack() {}, retry() {} };
  await consumeQueue({ messages: [message] }, env, { fetchImpl });
  assert.match(discordHeaders["user-agent"], /^DiscordBot \(/);
  assert.equal(discordURL, "https://discord.com/api/v10/webhooks/app-1/token/messages/@original");
});

test("malformed signatures are rejected instead of crashing", () => {
  assert.equal(verifyDiscordSignature("ab".repeat(32), "00", "1", "{}"), false);
  assert.equal(verifyDiscordSignature("ab", "00".repeat(64), "1", "{}"), false);
});

test("flagged prices never appear as the cheapest store", () => {
  const selection = selectPrices([
    { store_name: "Microsoft Store", price: 359, currency: "ARS", warning: "Precio inusualmente bajo" },
    { store_name: "Steam", price: 7700, currency: "ARS" },
  ], 1500);
  assert.equal(selection[0].store_name, "Steam");
  assert.ok(selection.every((price) => !price.warning));
});
