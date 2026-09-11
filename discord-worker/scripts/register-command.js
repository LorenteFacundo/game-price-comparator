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
  throw new Error(`No se pudo registrar /precio (${response.status}): ${await response.text()}`);
}

console.log("Comando global /precio registrado.");
