# Bot de Discord de PricePulse

`/precio juego:Hades` responde en el canal con los precios finales estimados para Argentina. Consulta exactamente `GET /api/search` de PricePulse: los precios publicados en ARS se conservan, los USD se convierten con dólar tarjeta e IVA y `0` se muestra como **Gratis**.

El Worker no usa Gateway ni base de datos. Discord recibe un acuse en menos de tres segundos y la consulta real viaja por una Cloudflare Queue, por lo que sigue funcionando aunque Render esté arrancando. Los términos buscados y los IDs de usuarios no se registran.

## Configuración única

1. Creá una aplicación en el [Discord Developer Portal](https://discord.com/developers/applications), agregale un bot y habilitá **Installation** para `Guild Install` con los scopes `applications.commands` y `bot`.
2. En **General Information**, copiá el `Application ID` y la `Public Key`. Conservá el token del bot como secreto.
3. Creá una cuenta de Cloudflare y ejecutá desde esta carpeta `npm install`, `npx wrangler login` y `npx wrangler queues create pricepulse-discord-jobs`.
4. Configurá las variables no secretas del Worker en Cloudflare: `BACKEND_URL`, `FRONTEND_URL`, `DISCORD_APPLICATION_ID` y `DISCORD_PUBLIC_KEY`. Usá las URLs públicas reales, sin barra final.
5. Desplegá con `npm run deploy`. Copiá `https://<tu-worker>/interactions` como **Interactions Endpoint URL** en Discord. Discord validará el PING automáticamente.
6. Registrá el comando con `DISCORD_APPLICATION_ID=... DISCORD_TOKEN=... npm run register-command`. El script usa `PUT` para dejar un solo comando global `/precio`, así que se puede ejecutar todas las veces que haga falta.
7. Usá la URL de instalación de Discord y agregá PricePulse a cualquier servidor. Los comandos globales pueden demorar unos minutos en aparecer.

## Despliegue continuo

En GitHub, cargá estos secretos: `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID` y `DISCORD_TOKEN`. Para que el workflow vuelva a registrar el comando en cada deploy, agregá además la variable de repositorio `DISCORD_APPLICATION_ID`. Las otras cuatro variables siguen configuradas directamente en Cloudflare, como en el paso 4.

Cuando cambie `discord-worker/` en `main`, el workflow ejecuta las pruebas, despliega el Worker y vuelve a registrar `/precio`. Si faltan los secretos, la verificación sigue corriendo y el despliegue queda omitido de forma segura.

## Desarrollo y verificación

```bash
cd discord-worker
npm install
npm run lint
npm test
```

La cola tiene dos reintentos para errores al editar el mensaje de Discord. Los errores de búsqueda se muestran amablemente en el mensaje original y nunca exponen detalles internos.
