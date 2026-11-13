import nacl from "tweetnacl";

export const DISCORD = {
  PING: 1,
  APPLICATION_COMMAND: 2,
  MESSAGE_COMPONENT: 3,
  PONG: 1,
  CHANNEL_MESSAGE: 4,
  DEFERRED_CHANNEL_MESSAGE: 5,
  DEFERRED_UPDATE: 6,
  EPHEMERAL: 1 << 6,
};

const encoder = new TextEncoder();
const decoder = new TextDecoder();
const PRIORITY_STORES = ["steam", "epic", "microsoft"];
const MAX_QUERY_BYTES = 50;
const MAX_CUSTOM_ID_LENGTH = 100;
export const DATA_CREDIT = "Precios de IsThereAnyDeal y Steam";

export function json(payload, status = 200) {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "content-type": "application/json; charset=utf-8" },
  });
}

function hexToBytes(value) {
  if (!/^[0-9a-f]+$/i.test(value) || value.length % 2 !== 0) return null;
  const bytes = new Uint8Array(value.length / 2);
  for (let index = 0; index < bytes.length; index += 1) {
    bytes[index] = Number.parseInt(value.slice(index * 2, index * 2 + 2), 16);
  }
  return bytes;
}

export function verifyDiscordSignature(publicKey, signature, timestamp, body) {
  const publicKeyBytes = hexToBytes(publicKey || "");
  const signatureBytes = hexToBytes(signature || "");
  if (!publicKeyBytes || !signatureBytes || !timestamp) return false;
  // tweetnacl lanza una excepción si la firma o la clave no tienen el largo
  // esperado; un pedido malformado debe responder 401, no 500.
  if (publicKeyBytes.length !== nacl.sign.publicKeyLength || signatureBytes.length !== nacl.sign.signatureLength) return false;
  return nacl.sign.detached.verify(
    encoder.encode(`${timestamp}${body}`),
    signatureBytes,
    publicKeyBytes,
  );
}

export function getInteractionUserId(interaction) {
  return interaction.member?.user?.id || interaction.user?.id || "";
}

export function getQuery(interaction) {
  return interaction.data?.options?.find((option) => option.name === "juego")?.value?.trim() || "";
}

export function validateQuery(query) {
  if (!query) return "Escribí el nombre de un juego para buscarlo.";
  if (encoder.encode(query).byteLength > MAX_QUERY_BYTES) {
    return "El nombre debe tener hasta 50 bytes para poder navegar los resultados.";
  }
  return "";
}

function base64UrlEncode(value) {
  const bytes = encoder.encode(value);
  let binary = "";
  bytes.forEach((byte) => { binary += String.fromCharCode(byte); });
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replaceAll("=", "");
}

function base64UrlDecode(value) {
  const padded = value.replaceAll("-", "+").replaceAll("_", "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const binary = atob(padded);
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
  return decoder.decode(bytes);
}

export function makeCursor(direction, index, requesterId, query) {
  const cursor = `p:${direction}:${index}:${requesterId}:${base64UrlEncode(query)}`;
  return cursor.length <= MAX_CUSTOM_ID_LENGTH ? cursor : "";
}

export function parseCursor(cursor) {
  const match = /^p:([pn]):(\d+):(\d+):([A-Za-z0-9_-]+)$/.exec(cursor || "");
  if (!match) return null;
  try {
    const query = base64UrlDecode(match[4]);
    if (validateQuery(query)) return null;
    return { direction: match[1], index: Number.parseInt(match[2], 10), requesterId: match[3], query };
  } catch {
    return null;
  }
}

export function rankMatches(results, query) {
  const needle = query.trim().toLocaleLowerCase("es-AR");
  const exact = [];
  const rest = [];
  for (const result of results || []) {
    if (result.title?.trim().toLocaleLowerCase("es-AR") === needle) exact.push(result);
    else rest.push(result);
  }
  return [...exact, ...rest];
}

function storeKind(storeName) {
  const normalized = (storeName || "").toLocaleLowerCase("en-US");
  return PRIORITY_STORES.find((store) => normalized.includes(store)) || "";
}

export function priceInARS(price, cardRate) {
  if (!Number.isFinite(price.price) || price.price < 0) return null;
  const currency = (price.currency || "").toUpperCase();
  if (currency === "ARS") return price.price;
  if (currency === "USD" && Number.isFinite(cardRate) && cardRate > 0) return price.price * cardRate;
  return null;
}

export function selectPrices(prices, cardRate) {
  const candidates = (prices || [])
    .map((price) => ({ ...price, finalARS: priceInARS(price, cardRate) }))
    // El backend marca con "warning" los precios absurdamente bajos (producto mal asociado).
    .filter((price) => price.finalARS !== null && !price.warning)
    .sort((left, right) => left.finalARS - right.finalARS || left.store_name?.localeCompare(right.store_name));

  const selected = [];
  const selectedKeys = new Set();
  const add = (price) => {
    if (!price) return;
    const key = `${price.store_name || ""}:${price.url || ""}`.toLocaleLowerCase("en-US");
    if (!selectedKeys.has(key) && selected.length < 6) {
      selected.push(price);
      selectedKeys.add(key);
    }
  };

  candidates.slice(0, 3).forEach(add);
  for (const store of PRIORITY_STORES) {
    add(candidates.find((price) => storeKind(price.store_name) === store));
  }
  return selected;
}

export function formatARS(value) {
  return new Intl.NumberFormat("es-AR", {
    style: "currency",
    currency: "ARS",
    maximumFractionDigits: 0,
  }).format(value);
}

export function validUrl(value) {
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
}

function priceLabel(price) {
  if (price.finalARS === 0) return "**Gratis**";
  return `**${formatARS(price.finalARS)}** · IVA incl.`;
}

function storeField(price, isCheapest) {
  const discount = Number.isFinite(price.discount_percent) && price.discount_percent > 0
    ? ` · -${price.discount_percent}%`
    : "";
  const destination = validUrl(price.url) ? `[Ver tienda](${price.url})` : "Sin enlace directo";
  return {
    name: `${isCheapest ? "Más barato · " : ""}${price.store_name || "Tienda"}`,
    value: `${priceLabel(price)}${discount}\n${destination}`,
    inline: true,
  };
}

function comparisonURL(frontendURL, query) {
  const url = new URL(frontendURL);
  url.searchParams.set("q", query);
  return url.toString();
}

export function renderResult({ response, game, query, resultIndex, requesterId, frontendURL }) {
  const prices = selectPrices(game.prices, response.card_rate);
  const resultCount = rankMatches(response.results, query).length;
  const cardRate = Number.isFinite(response.card_rate) && response.card_rate > 0 ? formatARS(response.card_rate) : "sin cotización válida";
  const tax = Number.isFinite(response.tax_rate) ? Math.round(response.tax_rate * 100) : 0;
  const updated = response.rates_updated_at || response.updated_at || "sin fecha";
  const webURL = comparisonURL(frontendURL, query);
  const warning = response.rates_stale ? " · cotización desactualizada" : "";
  const fields = prices.map((price, index) => storeField(price, index === 0));

  if (!fields.length) {
    fields.push({
      name: "Precios no convertibles",
      value: "No hay cotización válida para calcular el total en ARS + IVA. Probá de nuevo en unos minutos.",
    });
  }

  const previous = makeCursor("p", Math.max(resultIndex - 1, 0), requesterId, query);
  const next = makeCursor("n", resultIndex + 1, requesterId, query);
  return {
    content: "",
    embeds: [{
      title: game.title || query,
      url: webURL,
      description: `[Abrir comparación completa](${webURL}) · Resultado ${resultIndex + 1} de ${resultCount}`,
      thumbnail: validUrl(game.image_url) ? { url: game.image_url } : undefined,
      fields,
      footer: { text: `ARS + IVA · dólar tarjeta ${cardRate} · IVA ${tax}% · actualizado ${updated}${warning}
${DATA_CREDIT}` },
    }],
    components: [{
      type: 1,
      components: [
        { type: 2, style: 2, custom_id: previous, label: "← Anterior", disabled: resultIndex <= 0 || !previous },
        { type: 2, style: 2, custom_id: next, label: "Siguiente →", disabled: resultIndex >= resultCount - 1 || !next },
      ],
    }],
    allowed_mentions: { parse: [] },
  };
}

export function renderFailure(message) {
  return { content: message, embeds: [], components: [], allowed_mentions: { parse: [] } };
}

export async function fetchSearch(backendURL, query, fetchImpl = fetch) {
  const url = new URL("/api/search", backendURL);
  url.searchParams.set("q", query);
  const response = await fetchImpl(url, { signal: AbortSignal.timeout(90_000) });
  if (!response.ok) throw new Error(`backend_${response.status}`);
  const data = await response.json();
  if (!Array.isArray(data.results)) throw new Error("backend_invalid_response");
  return data;
}
