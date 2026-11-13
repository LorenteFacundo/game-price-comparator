import {
  DISCORD,
  fetchSearch,
  getInteractionUserId,
  getQuery,
  json,
  parseCursor,
  rankMatches,
  renderFailure,
  renderResult,
  validateQuery,
  verifyDiscordSignature,
} from "./pricepulse.js";

const USER_AGENT = "DiscordBot (https://github.com/LorenteFacundo/game-price-comparator, 1.0)";

const PRIVATE = (content) => json({ type: DISCORD.CHANNEL_MESSAGE, data: { content, flags: DISCORD.EPHEMERAL } });

function validWorkerConfig(env) {
  return env.BACKEND_URL && env.FRONTEND_URL && env.DISCORD_APPLICATION_ID;
}

async function enqueue(env, job) {
  await env.DISCORD_JOBS.send(job);
}

export async function handleInteraction(request, env, { verify = verifyDiscordSignature } = {}) {
  const url = new URL(request.url);
  if (url.pathname !== "/interactions") return new Response("Not found", { status: 404 });
  if (request.method !== "POST") return new Response("Method not allowed", { status: 405, headers: { allow: "POST" } });

  const body = await request.text();
  if (!verify(env.DISCORD_PUBLIC_KEY, request.headers.get("x-signature-ed25519"), request.headers.get("x-signature-timestamp"), body)) {
    return new Response("Invalid request signature", { status: 401 });
  }

  let interaction;
  try {
    interaction = JSON.parse(body);
  } catch {
    return new Response("Invalid JSON", { status: 400 });
  }

  if (interaction.type === DISCORD.PING) return json({ type: DISCORD.PONG });
  if (!validWorkerConfig(env)) return PRIVATE("PricePulse no está configurado todavía. Avisale al administrador del bot.");

  if (interaction.type === DISCORD.APPLICATION_COMMAND) {
    if (interaction.data?.name !== "precio") return PRIVATE("Ese comando no existe en PricePulse.");
    const query = getQuery(interaction);
    const error = validateQuery(query);
    if (error) return PRIVATE(error);
    await enqueue(env, {
      application_id: interaction.application_id,
      interaction_token: interaction.token,
      query,
      result_index: 0,
      requester_id: getInteractionUserId(interaction),
      update_type: "command",
    });
    return json({ type: DISCORD.DEFERRED_CHANNEL_MESSAGE });
  }

  if (interaction.type === DISCORD.MESSAGE_COMPONENT) {
    const cursor = parseCursor(interaction.data?.custom_id);
    if (!cursor) return PRIVATE("Este botón ya no es válido. Usá /precio otra vez.");
    if (cursor.requesterId !== getInteractionUserId(interaction)) {
      return PRIVATE("Sólo quien inició esta búsqueda puede cambiar el resultado.");
    }
    await enqueue(env, {
      application_id: interaction.application_id,
      interaction_token: interaction.token,
      query: cursor.query,
      result_index: cursor.index,
      requester_id: cursor.requesterId,
      update_type: "navigation",
    });
    return json({ type: DISCORD.DEFERRED_UPDATE });
  }

  return PRIVATE("PricePulse no puede procesar esta interacción.");
}

async function editOriginal(job, payload, fetchImpl = fetch) {
  const endpoint = `https://discord.com/api/v10/webhooks/${job.application_id}/interactions/${job.interaction_token}/messages/@original`;
  const response = await fetchImpl(endpoint, {
    method: "PATCH",
    // Discord exige un User-Agent identificable; los Workers no envían uno por
    // defecto y algunas respuestas terminan bloqueadas sin él.
    headers: { "content-type": "application/json", "user-agent": USER_AGENT },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    const detail = (await response.text().catch(() => "")).slice(0, 300);
    throw new Error(`discord_${response.status} ${detail}`.trim());
  }
}

export async function processJob(job, env, { fetchImpl = fetch } = {}) {
  let payload;
  try {
    const response = await fetchSearch(env.BACKEND_URL, job.query, fetchImpl);
    const matches = rankMatches(response.results, job.query);
    const index = Math.min(Math.max(job.result_index, 0), Math.max(matches.length - 1, 0));
    const game = matches[index];
    payload = game
      ? renderResult({ response, game, query: job.query, resultIndex: index, requesterId: job.requester_id, frontendURL: env.FRONTEND_URL })
      : renderFailure(`No encontré coincidencias para **${job.query}**. Probá con otro nombre.`);
  } catch (error) {
    // Se registra sólo el motivo técnico; nunca la búsqueda ni el usuario.
    console.error("pricepulse_backend_error", error?.message);
    payload = renderFailure("No pude consultar los precios ahora. PricePulse va a reintentar si Discord no recibió esta actualización; si persiste, probá nuevamente en un minuto.");
  }
  await editOriginal(job, payload, fetchImpl);
}

export async function consumeQueue(batch, env, options = {}) {
  for (const message of batch.messages) {
    try {
      await processJob(message.body, env, options);
      message.ack();
    } catch (error) {
      console.error("pricepulse_discord_edit_error", error?.message, `attempt=${message.attempts ?? "?"}`);
      message.retry();
    }
  }
}

export default {
  fetch(request, env) {
    return handleInteraction(request, env);
  },
  queue(batch, env) {
    return consumeQueue(batch, env);
  },
};
