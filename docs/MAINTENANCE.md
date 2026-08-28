# Mantenimiento de PricePulse

Esta guía mantiene el proyecto operable sin convertirlo en un trabajo de soporte de tiempo completo.

## Rutina

### Una vez por semana

1. Revisar los Issues nuevos y etiquetarlos como `precio incorrecto`, `error` o `mejora`.
2. Mirar los logs del backend buscando respuestas 5xx, tiempos altos o proveedores que fallen.
3. Abrir la web desde móvil y comprobar una búsqueda, una oferta y la conversión ARS + IVA.

### Una vez por mes

1. Revisar y fusionar las actualizaciones de Dependabot que pasen CI.
2. Ejecutar localmente `go test ./...`, `go vet ./...`, `npm run lint`, `npm run build` y `npm run test:ui`.
3. Cerrar o priorizar el feedback acumulado.
4. Confirmar que las variables de producción siguen configuradas y que las claves no están por vencer.

## Monitoreo

Configurar un monitor HTTP para consultar cada cinco minutos:

```text
GET https://TU-BACKEND/api/health
```

La respuesta correcta es HTTP 200 con `status: ok`. El endpoint no consulta proveedores externos ni consume sus cuotas. Sirve para detectar que el proceso y la red están disponibles.

Para diagnosticar una petición fallida, buscar en los logs su header `X-Request-ID`. Los logs registran método y ruta, pero deliberadamente omiten query strings y búsquedas de usuarios.

## Incidentes

1. Confirmar si falla sólo una fuente o toda la API desde el panel **Estado**.
2. Revisar el último despliegue y los logs asociados al request ID.
3. Si el problema empezó después de un despliegue, volver al último commit estable desde el proveedor de hosting.
4. Si falla una API externa, mantener la web disponible y mostrar el aviso existente; no eliminar datos ni alterar precios para ocultarlo.
5. Documentar la causa y agregar una prueba antes del siguiente despliegue.

## Feedback y privacidad

Los reportes se crean en GitHub y son públicos. La interfaz advierte que no deben incluir datos personales. PricePulse no tiene cuentas: favoritos e historial permanecen en el navegador del usuario.

Las tarjetas de precio adjuntan automáticamente juego, tienda, precio mostrado, enlace fuente y URL de la página. Nunca se envían claves, cookies ni contenido del almacenamiento local.

## Publicar una versión

1. Verificar que la rama esté limpia y actualizada.
2. Ejecutar todas las pruebas.
3. Escribir un commit descriptivo y subirlo a `main`.
4. Confirmar el despliegue del frontend y backend.
5. Consultar `/api/health` y hacer una búsqueda real.
6. Agregar el cambio visible a la lista de novedades de `SupportDialog.jsx`.
