import { useState } from 'react'

// Imagen de portada con respaldo: si la URL falla (CDN caído, bloqueo de red,
// juego sin arte) mostramos un placeholder en vez del ícono de imagen rota.
export default function GameImage({ src, fallback = '✦' }) {
  const [failedSrc, setFailedSrc] = useState('')
  if (!src || failedSrc === src) return <div className="image-fallback" aria-hidden="true">{fallback}</div>
  return <img src={src} alt="" width="300" height="140" loading="lazy" onError={() => setFailedSrc(src)} />
}
