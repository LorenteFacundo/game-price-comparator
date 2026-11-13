const applicationId = process.env.DISCORD_APPLICATION_ID;
const token = process.env.DISCORD_TOKEN;

if (!applicationId || !token) {
  throw new Error("DISCORD_APPLICATION_ID y DISCORD_TOKEN son obligatorios.");
}

const commands = [{
  name: "precio",
  description: "Compara precios finales estimados para Argentina",
  type: 1,
  options: [{
    name: "juego",
    description: "Nombre del juego",
    type: 3,
    required: true,
    max_length: 50,
  }],
}, {
  name: "ofertas",
  description: "Las mejores ofertas del momento en Steam, Epic y Microsoft",
  type: 1,
}, {
  name: "gratis",
  description: "Juegos gratis por tiempo limitado",
  type: 1,
}, {
  name: "populares",
  description: "Los más vendidos de Steam con su precio y dónde está más barato",
  type: 1,
}, {
  name: "ayuda",
  description: "Qué hace PricePulse y cómo usarlo",
  type: 1,
}];

const response = await fetch(`https://discord.com/api/v10/applications/${applicationId}/commands`, {
  method: "PUT",
  headers: {
    authorization: `Bot ${token}`,
    "content-type": "application/json",
  },
  body: JSON.stringify(commands),
});

if (!response.ok) {
  throw new Error(`No se pudieron registrar los comandos (${response.status}): ${await response.text()}`);
}

console.log(`Comandos globales registrados: ${commands.map((command) => `/${command.name}`).join(", ")}.`);
