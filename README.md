# PricePulse

Comparador de ofertas de videojuegos pensado para Argentina. Prioriza tiendas verificadas, conserva la moneda original que devuelve cada proveedor y permite comparar ARS y USD de forma transparente.

## Qué incluye

- Búsqueda de juegos en IsThereAnyDeal con precios publicados para Argentina.
- Ofertas de Steam, Epic y Microsoft ordenadas por popularidad, recepción y jugadores, sin perder los juegos gratis.
- Selector USD / ARS + IMP aplicado tanto a comparaciones como a ofertas.
- Ranking de precios que solo compara monedas convertibles conocidas; nunca interpreta USD como ARS.
- Favoritos e historial de búsqueda guardados localmente en el navegador.
- Interfaz responsive, accesible y sin promoción de cuentas compartidas.
- Caché en memoria limitada para proteger las cuotas de proveedores en despliegues gratuitos.
- Reportes contextuales de precios y sugerencias mediante GitHub Issues.
- Estado operativo, versión desplegada y fecha real de actualización de los datos.

## Ejecutar localmente

1. Copiá `backend/.env.example` a `backend/.env` y completá `ITAD_API_KEY`.
2. En una terminal, ejecutá `cd backend && go run .`.
3. En otra, ejecutá `cd frontend && npm install && npm run dev`.
4. Abrí `http://localhost:5173`.

Para usar un backend remoto en el frontend, copiá `frontend/.env.example` a `frontend/.env` y configurá `VITE_API_URL` antes de compilar.

## Mantenimiento en producción

- `GET /api/health` informa estado, versión, uptime y proveedores configurados. Se puede conectar a cualquier monitor HTTP.
- La API conserva resultados parciales si una fuente de ofertas falla y avisa cuando la cotización usada está desactualizada.
- Cada request del backend registra ruta, estado, duración e ID de seguimiento sin guardar los términos buscados.
- Los usuarios pueden reportar un precio desde su tarjeta; el contexto se completa automáticamente en GitHub.
- CI valida backend, frontend y experiencia Playwright antes de integrar cambios.
- El bot público de Discord permite usar `/precio juego:Hades` en cualquier servidor y reutiliza la misma comparación ARS + IVA de la web.
- Dependabot propone actualizaciones mensuales de Go, npm y GitHub Actions.

La rutina recomendada y el procedimiento ante incidentes están en [docs/MAINTENANCE.md](docs/MAINTENANCE.md).
La configuración única del bot está en [discord-worker/README.md](discord-worker/README.md).

## Verificación

```bash
cd backend
go test ./...
go vet ./...

cd ../frontend
npm run lint
npm run build
npm run test:ui

cd ../discord-worker
npm ci
npm run lint
npm test
```

## Arquitectura actual

`React/Vite → API Go → ITAD + Steam + Bluelytics`

El backend está preparado para evolucionar a una base persistente de snapshots, alertas y listas de deseos. Por ahora la caché es intencionalmente simple para que siga siendo fácil de desplegar en servicios gratuitos.
