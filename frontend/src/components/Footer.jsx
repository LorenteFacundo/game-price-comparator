export default function Footer({ health, updates, onOpenSupport }) {
	const latest = [updates.search, updates.deals, updates.discover].filter(Boolean).sort((a, b) => new Date(b) - new Date(a))[0]
	const updated = latest ? new Intl.DateTimeFormat('es-AR', { hour: '2-digit', minute: '2-digit' }).format(new Date(latest)) : '—'
  return <footer className="site-footer">
    <div className="footer-identity"><span>PRICEPULSE · ARGENTINA</span><small>Precios orientativos. El total válido es el de la tienda.</small></div>
    <div className="footer-status"><span className={`status-dot ${health.state === 'online' ? 'is-online' : ''}`} aria-hidden="true" />{health.state === 'online' ? 'API online' : health.state === 'checking' ? 'Comprobando' : 'Revisar estado'}<small>Actualizado {updated}</small></div>
    <nav className="footer-links" aria-label="Ayuda y proyecto"><button type="button" onClick={() => onOpenSupport('feedback')}>Contacto</button><button type="button" onClick={() => onOpenSupport('status')}>Estado</button><button type="button" onClick={() => onOpenSupport('changes')}>Novedades</button><a href="https://github.com/LorenteFacundo/game-price-comparator" target="_blank" rel="noopener noreferrer">GitHub ↗</a></nav>
  </footer>
}
