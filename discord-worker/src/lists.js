// Comandos de listas: /ofertas, /gratis, /populares y /ayuda.
// Reutilizan los endpoints de la web (/api/deals y /api/discover), así el bot
// muestra exactamente lo mismo que la portada.
import { formatARS, priceInARS, validUrl } from "./pricepulse.js";

export const LIST_COMMANDS = ["ofertas", "gratis", "populares"];
const MAX_ITEMS = 8;
const EMBED_COLOR = 0xcdfc53;

export async function fetchBackendJSON(backendURL, path, fetchImpl = fetch) {
  const url = new URL(path, backendURL);
  const response = await fetchImpl(url, { signal: AbortSignal.timeout(90_000) });
  if (!response.ok) throw new Error(`backend_${response.status}`);
  return response.json();
}

function finalPrice(price, cardRate) {
  if (!price) return "";
  const ars = priceInARS(price, cardRate);
  if (ars === 0) return "**Gratis**";
  if (ars !== null) return `**${formatARS(ars)}**`;
  return `**${price.currency || ""} ${price.price}**`.trim();
}

function link(label, url) {
  return validUrl(url) ? `[${label}](${url})` : "";
}

function shortDate(value) {
  const date = new Date(value || "");
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("es-AR", { day: "numeric", month: "numeric", timeZone: "America/Argentina/Buenos_Aires" }).format(date);
}

function footerText(response) {
  const cardRate = Number(response.card_rate);
  const tax = Number.isFinite(response.tax_rate) ? Math.round(response.tax_rate * 100) : 21;
  if (!(cardRate > 0)) return "Precios publicados por cada tienda · cotización no disponible";
  return `ARS con IVA ${tax}% · dólar tarjeta ${formatARS(cardRate)}${response.rates_stale ? " · cotización desactualizada" : ""}`;
}

function webButton(frontendURL, label = "Ver en PricePulse") {
  if (!validUrl(frontendURL)) return [];
  return [{ type: 1, components: [{ type: 2, style: 5, label, url: frontendURL }] }];
}

function listMessage({ title, description, items, thumbnail, response, frontendURL }) {
  return {
    content: "",
    embeds: [{
      title,
      url: validUrl(frontendURL) ? frontendURL : undefined,
      color: EMBED_COLOR,
      description: [description, "", ...items].join("\n").slice(0, 4000),
      thumbnail: validUrl(thumbnail) ? { url: thumbnail } : undefined,
      footer: { text: footerText(response) },
    }],
    components: webButton(frontendURL),
    allowed_mentions: { parse: [] },
  };
}

export function renderDeals(response, frontendURL) {
  // Primero las destacadas (ranking por popularidad y reseñas) y, si son pocas,
  // completamos con los mayores descuentos sin repetir juegos.
  const seen = new Set();
  const byScore = [...(response.discounts || [])].sort((left, right) => (right.score || 0) - (left.score || 0));
  const deals = [...(response.featured || []), ...byScore].filter((deal) => {
    const key = (deal.title || "").toLocaleLowerCase("es-AR");
    if (!(deal.price > 0) || seen.has(key)) return false;
    seen.add(key);
    return true;
  }).slice(0, MAX_ITEMS);
  if (!deals.length) return listMessage({ title: "🔥 Ofertas destacadas", description: "No encontramos ofertas destacadas en este momento.", items: [], response, frontendURL });
  const items = deals.map((deal, index) => {
    const discount = deal.discount_percent > 0 ? ` (-${deal.discount_percent}%)` : "";
    const store = link(deal.store_name || "Tienda", deal.url) || deal.store_name || "Tienda";
    return `**${index + 1}. ${deal.title}** — ${finalPrice(deal, response.card_rate)}${discount} en ${store}`;
  });
  return listMessage({ title: "🔥 Ofertas destacadas", description: "Las mejores ofertas de Steam, Epic y Microsoft, ordenadas por popularidad y reseñas.", items, thumbnail: deals[0].image_url, response, frontendURL });
}

export function renderFree(response, frontendURL) {
  const seen = new Set();
  const free = (response.free || []).filter((deal) => {
    const key = `${deal.title}|${deal.store_name}`.toLocaleLowerCase("es-AR");
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  }).slice(0, MAX_ITEMS);
  if (!free.length) return listMessage({ title: "🎁 Juegos gratis", description: "Ahora no hay juegos gratis por tiempo limitado en Steam, Epic ni Microsoft. ¡Volvé a fijarte en unos días!", items: [], response, frontendURL });
  const items = free.map((deal) => {
    const until = shortDate(deal.expires_at);
    const store = link(`Conseguir en ${deal.store_name || "la tienda"}`, deal.url) || deal.store_name || "";
    return `**${deal.title}** — ${store}${until ? ` · hasta el ${until}` : ""}`;
  });
  return listMessage({ title: "🎁 Juegos gratis", description: "Gratis por tiempo limitado: si lo agregás a tu cuenta, queda tuyo.", items, thumbnail: free[0].image_url, response, frontendURL });
}

export function renderPopular(response, frontendURL) {
  const games = (response.popular || []).slice(0, MAX_ITEMS);
  if (!games.length) return listMessage({ title: "📈 Populares en Steam", description: "No pudimos cargar el ranking de Steam ahora.", items: [], response, frontendURL });
  const items = games.map((game) => {
    const steam = game.steam_price ? `Steam ${finalPrice(game.steam_price, response.card_rate)}` : "Sin precio publicado";
    const steamARS = game.steam_price ? priceInARS(game.steam_price, response.card_rate) : null;
    const bestARS = game.best_price ? priceInARS(game.best_price, response.card_rate) : null;
    const best = steamARS !== null && bestARS !== null && bestARS < steamARS
      ? ` · más barato ${finalPrice(game.best_price, response.card_rate)} en ${game.best_price.store_name}`
      : "";
    return `**#${game.rank} ${game.title}** — ${steam}${best}`;
  });
  return listMessage({ title: "📈 Populares en Steam", description: "Los más vendidos de Steam ahora, con su precio y dónde está más barato. Usá `/precio` para ver todas las tiendas.", items, thumbnail: games[0].image_url, response, frontendURL });
}

export async function buildListMessage(kind, env, fetchImpl = fetch) {
  if (kind === "populares") return renderPopular(await fetchBackendJSON(env.BACKEND_URL, "/api/discover", fetchImpl), env.FRONTEND_URL);
  const deals = await fetchBackendJSON(env.BACKEND_URL, "/api/deals?limit=12", fetchImpl);
  return kind === "gratis" ? renderFree(deals, env.FRONTEND_URL) : renderDeals(deals, env.FRONTEND_URL);
}

export function helpMessage(frontendURL) {
  return {
    embeds: [{
      title: "PricePulse · ayuda",
      color: EMBED_COLOR,
      description: [
        "Compará precios de juegos en Steam, Epic y Microsoft Store, en pesos con impuestos incluidos.",
        "",
        "`/precio juego:Hades` — precio en cada tienda y cuál es la más barata",
        "`/ofertas` — las mejores ofertas del momento",
        "`/gratis` — juegos gratis por tiempo limitado",
        "`/populares` — los más vendidos de Steam con su precio",
        "",
        "Los precios en USD se convierten con el dólar tarjeta + IVA. Verificá siempre el total en la tienda antes de comprar.",
      ].join("\n"),
    }],
    components: webButton(frontendURL, "Abrir PricePulse"),
    allowed_mentions: { parse: [] },
  };
}
