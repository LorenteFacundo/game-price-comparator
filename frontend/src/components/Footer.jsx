const links = {
  repository: 'https://github.com/LorenteFacundo/game-price-comparator',
  github: 'https://github.com/LorenteFacundo',
  linkedin: 'https://www.linkedin.com/in/facundo-lorente-692206230',
  discordBot: 'https://discord.com/oauth2/authorize?client_id=1548062341973942315',
  itad: 'https://isthereanydeal.com/',
  steam: 'https://store.steampowered.com/',
  bluelytics: 'https://bluelytics.com.ar/',
}

function ExternalLink({ href, children }) {
  return <a href={href} target="_blank" rel="noopener noreferrer">{children}</a>
}

export default function Footer({ health, updates, onOpenSupport }) {
  const latest = [updates.search, updates.deals, updates.discover].filter(Boolean).sort((a, b) => new Date(b) - new Date(a))[0]
  const updated = latest ? new Intl.DateTimeFormat('es-AR', { hour: '2-digit', minute: '2-digit' }).format(new Date(latest)) : '—'
  return <footer className="site-footer">
    <div className="footer-identity"><span>PRICEPULSE · ARGENTINA</span><small>Precios orientativos. El total válido es el de la tienda.</small></div>
    <div className="footer-status"><span className={`status-dot ${health.state === 'online' ? 'is-online' : ''}`} aria-hidden="true" />{health.state === 'online' ? 'API online' : health.state === 'checking' ? 'Comprobando' : 'Revisar estado'}<small>Actualizado {updated}</small></div>
    <nav className="footer-links" aria-label="Ayuda y proyecto"><button type="button" onClick={() => onOpenSupport('feedback')}>Contacto</button><button type="button" onClick={() => onOpenSupport('status')}>Estado</button><button type="button" onClick={() => onOpenSupport('changes')}>Novedades</button><ExternalLink href={links.discordBot}>Bot de Discord ↗</ExternalLink><ExternalLink href={links.repository}>Código ↗</ExternalLink></nav>
    <div className="footer-credits">
      <p>Precios de <ExternalLink href={links.itad}>IsThereAnyDeal</ExternalLink> y <ExternalLink href={links.steam}>Steam</ExternalLink>. Cotización de <ExternalLink href={links.bluelytics}>Bluelytics</ExternalLink>. PricePulse no está afiliado a ninguna de estas tiendas ni servicios.</p>
      <p>Hecho por Facundo Lorente · <ExternalLink href={links.github}>GitHub</ExternalLink> · <ExternalLink href={links.linkedin}>LinkedIn</ExternalLink></p>
    </div>
  </footer>
}
