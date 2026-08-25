# PricePulse

Comparador de ofertas de videojuegos pensado para Argentina. Prioriza tiendas verificadas, conserva la moneda original que devuelve cada proveedor y permite comparar ARS y USD de forma transparente.

## Qué incluye

- Búsqueda de juegos en IsThereAnyDeal y precio regional/global de Steam para el resultado principal.
- Ofertas destacadas reales de ITAD, ordenadas por descuento.
- Ranking de precios que solo compara monedas convertibles conocidas; nunca interpreta USD como ARS.
- Favoritos e historial de búsqueda guardados localmente en el navegador.
- Interfaz responsive, accesible y sin promoción de cuentas compartidas.
- Caché en memoria para proteger las cuotas de proveedores en despliegues gratuitos.

## Ejecutar localmente

1. Copiá `backend/.env.example` a `backend/.env` y completá `ITAD_API_KEY`.
2. En una terminal, ejecutá `cd backend && go run .`.
3. En otra, ejecutá `cd frontend && npm install && npm run dev`.
4. Abrí `http://localhost:5173`.

Para usar un backend remoto en el frontend, copiá `frontend/.env.example` a `frontend/.env` y configurá `VITE_API_URL` antes de compilar.

## Verificación

```bash
cd backend
go test ./...
go vet ./...

cd ../frontend
npm run lint
npm run build
npm run test:ui
```

## Arquitectura actual

`React/Vite → API Go → ITAD + Steam + Bluelytics`

El backend está preparado para evolucionar a una base persistente de snapshots, alertas y listas de deseos. Por ahora la caché es intencionalmente simple para que siga siendo fácil de desplegar en servicios gratuitos.
