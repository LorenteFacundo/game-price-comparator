import { useEffect, useRef, useState } from 'react'

const repositoryURL = (import.meta.env.VITE_FEEDBACK_REPO_URL || 'https://github.com/LorenteFacundo/game-price-comparator').replace(/\/$/, '')

const changes = [
  { date: '28 AGO 2026', title: 'Centro de soporte', detail: 'Reportes de precios, estado del servicio y canal de mejoras desde la app.' },
  { date: '26 AGO 2026', title: 'Ofertas top tier', detail: 'Más peso para popularidad, jugadores y recepción; los descuentos chicos siguen separados.' },
  { date: '26 AGO 2026', title: 'Precios para Argentina', detail: 'USD directo o conversión ARS + IVA, sin mezclar precios globales.' },
]

function formatTimestamp(value) {
  if (!value) return 'Todavía sin datos'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Todavía sin datos'
  return new Intl.DateTimeFormat('es-AR', { dateStyle: 'short', timeStyle: 'short' }).format(date)
}

function buildIssueURL(kind, message, context) {
  const labels = kind === 'price' ? 'precio incorrecto' : kind === 'bug' ? 'error' : 'mejora'
  const subject = context?.gameTitle || (kind === 'price' ? 'Precio incorrecto' : kind === 'bug' ? 'Error en la web' : 'Sugerencia')
  const contextLines = context ? [
    `- Juego: ${context.gameTitle || 'No indicado'}`,
    `- Tienda: ${context.store || 'No indicada'}`,
    `- Precio mostrado: ${context.displayedPrice || 'No indicado'}`,
    `- Fuente: ${context.sourceURL || 'No indicada'}`,
  ] : []
  const body = [
    '## Qué pasó o qué mejorarías',
    message.trim() || 'El precio mostrado no coincide con la tienda.',
    '',
    ...(contextLines.length ? ['## Contexto automático', ...contextLines, ''] : []),
    '## Página',
    window.location.href,
    '',
    '_Reporte generado desde PricePulse. No incluye datos personales._',
  ].join('\n')
  const issueURL = new URL(`${repositoryURL}/issues/new`)
  issueURL.searchParams.set('title', `[${labels}] ${subject}`)
  issueURL.searchParams.set('body', body)
  issueURL.searchParams.set('labels', labels)
  return issueURL.toString()
}

export default function SupportDialog({ open, initialView, context, health, lastUpdated, warnings, onClose }) {
  if (!open) return null
  return <SupportDialogContent initialView={initialView} context={context} health={health} lastUpdated={lastUpdated} warnings={warnings} onClose={onClose} />
}

function SupportDialogContent({ initialView, context, health, lastUpdated, warnings, onClose }) {
  const dialog = useRef(null)
  const closeButton = useRef(null)
  const [view, setView] = useState(initialView || 'feedback')
  const [kind, setKind] = useState(context ? 'price' : 'idea')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    const previousFocus = document.activeElement
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const timer = window.setTimeout(() => closeButton.current?.focus(), 0)
    const onKeyDown = (event) => {
      if (event.key === 'Escape') onClose()
      if (event.key === 'Tab') {
        const focusable = [...dialog.current.querySelectorAll('button, select, textarea, a[href]')].filter((element) => !element.disabled)
        const first = focusable[0]
        const last = focusable[focusable.length - 1]
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault()
          last?.focus()
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault()
          first?.focus()
        }
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.clearTimeout(timer)
      window.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = previousOverflow
      previousFocus?.focus()
    }
  }, [onClose])

  function submitFeedback(event) {
    event.preventDefault()
    if (!context && message.trim().length < 8) {
      setError('Contanos un poco más para poder entenderlo.')
      return
    }
    window.open(buildIssueURL(kind, message, context), '_blank', 'noopener,noreferrer')
    onClose()
  }

  const healthOnline = health.state === 'online'
  return (
    <div className="support-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section ref={dialog} className="support-dialog" role="dialog" aria-modal="true" aria-labelledby="support-title">
        <header className="support-dialog-header">
          <div><span className="eyebrow">Mantenimiento</span><h2 id="support-title">Centro de soporte</h2></div>
          <button ref={closeButton} className="support-close" type="button" onClick={onClose} aria-label="Cerrar">×</button>
        </header>
        <nav className="support-tabs" aria-label="Secciones de soporte">
          {[['feedback', 'Contacto'], ['status', 'Estado'], ['changes', 'Novedades']].map(([id, label]) => <button type="button" key={id} className={view === id ? 'is-active' : ''} onClick={() => setView(id)}>{label}</button>)}
        </nav>

        {view === 'feedback' && <form className="feedback-form" onSubmit={submitFeedback}>
          {context && <div className="report-context"><span>Reporte preparado</span><strong>{context.gameTitle}</strong><small>{context.store} · {context.displayedPrice}</small></div>}
          <label>Tipo<select value={kind} onChange={(event) => setKind(event.target.value)}><option value="price">Precio incorrecto</option><option value="bug">Algo no funciona</option><option value="idea">Idea o mejora</option></select></label>
          <label>Mensaje<textarea value={message} onChange={(event) => { setMessage(event.target.value); setError('') }} rows="5" maxLength="1500" placeholder={context ? '¿Qué precio viste en la tienda?' : 'Contanos qué pasó o qué te gustaría ver…'} /></label>
          {error && <p className="feedback-error" role="alert">{error}</p>}
          <p className="privacy-note">Se abrirá un reporte público en GitHub. No escribas datos personales.</p>
          <button className="feedback-submit" type="submit">Preparar reporte <span aria-hidden="true">↗</span></button>
        </form>}

        {view === 'status' && <div className="status-panel">
          <div className={`system-state ${healthOnline ? 'is-online' : health.state === 'checking' ? 'is-checking' : 'is-offline'}`}><span aria-hidden="true" /><div><strong>{healthOnline ? 'API funcionando' : health.state === 'checking' ? 'Comprobando API' : 'API sin respuesta'}</strong><small>{healthOnline ? `Versión ${health.data?.version || 'actual'}` : health.message || 'Reintentaremos al recargar.'}</small></div></div>
          <dl><div><dt>Datos actualizados</dt><dd>{formatTimestamp(lastUpdated)}</dd></div><div><dt>Fuentes</dt><dd>Steam · Epic · Microsoft · ITAD</dd></div><div><dt>Avisos activos</dt><dd>{warnings.length}</dd></div></dl>
          {warnings.length > 0 && <ul className="status-warnings">{warnings.map((warning) => <li key={warning}>{warning}</li>)}</ul>}
          <p className="privacy-note">Los precios pueden cambiar antes de la próxima actualización. Confirmá siempre el total en la tienda.</p>
        </div>}

        {view === 'changes' && <ol className="changes-list">{changes.map((change) => <li key={`${change.date}-${change.title}`}><time>{change.date}</time><div><strong>{change.title}</strong><p>{change.detail}</p></div></li>)}</ol>}
      </section>
    </div>
  )
}
