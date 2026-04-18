const STORES = [
  {
    name: 'Steam',
    color: '#1b2838',
    accent: '#66c0f4',
    icon: '🎮',
    search: (q) => `https://store.steampowered.com/search/?term=${encodeURIComponent(q)}`,
  },
  {
    name: 'Instant Gaming',
    color: '#1a1a2e',
    accent: '#e94560',
    icon: '⚡',
    search: (q) => `https://www.instant-gaming.com/es/busqueda/?q=${encodeURIComponent(q)}`,
  },
  {
    name: 'Eneba',
    color: '#0f1923',
    accent: '#00e5a0',
    icon: '🛒',
    search: (q) => `https://www.eneba.com/store/all?text=${encodeURIComponent(q)}`,
  },
  {
    name: 'G2A',
    color: '#121212',
    accent: '#f7a600',
    icon: '🔑',
    search: (q) => `https://www.g2a.com/search?query=${encodeURIComponent(q)}`,
  },
  {
    name: 'MundoSteam',
    color: '#1a0a0a',
    accent: '#ff4444',
    icon: '⚠️',
    warning: 'Vende acceso a cuentas compartidas, no el juego en tu cuenta. Usá bajo tu propio riesgo.',
    search: (q) => `https://mundosteam.com/`,
  },
]

export default function StoreButtons({ query }) {
  return (
    <div style={{ width: '100%', maxWidth: '700px' }}>
      <p style={{ fontSize: '12px', color: 'var(--muted)', marginBottom: '10px' }}>
        Buscar directamente en cada tienda
      </p>
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))',
        gap: '10px',
      }}>
        {STORES.map(store => (
          <StoreButton key={store.name} store={store} query={query} />
        ))}
      </div>
    </div>
  )
}

function StoreButton({ store, query }) {
  const [showTooltip, setShowTooltip] = React.useState(false)

  const handleClick = () => {
    if (store.warning) {
      if (!window.confirm(`⚠️ Advertencia: ${store.warning}\n\n¿Querés continuar?`)) return
    }
    window.open(store.search(query || ''), '_blank')
  }

  return (
    <div style={{ position: 'relative' }}>
      <button
        onClick={handleClick}
        onMouseEnter={() => store.warning && setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        style={{
          width: '100%',
          background: store.color,
          border: `1px solid ${store.accent}33`,
          borderRadius: '10px',
          padding: '14px 10px',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: '8px',
          cursor: 'pointer',
          transition: 'border-color 0.2s, transform 0.1s',
        }}
        onMouseOver={e => {
          e.currentTarget.style.borderColor = store.accent
          e.currentTarget.style.transform = 'translateY(-2px)'
        }}
        onMouseOut={e => {
          e.currentTarget.style.borderColor = `${store.accent}33`
          e.currentTarget.style.transform = 'translateY(0)'
        }}
      >
        <span style={{ fontSize: '22px' }}>{store.icon}</span>
        <span style={{
          fontSize: '12px',
          fontWeight: 600,
          color: store.accent,
          textAlign: 'center',
          lineHeight: 1.2,
        }}>
          {store.name}
        </span>
        {store.warning && (
          <span style={{
            fontSize: '9px',
            background: 'rgba(255,68,68,0.15)',
            color: '#ff4444',
            padding: '2px 6px',
            borderRadius: '10px',
            fontWeight: 600,
          }}>
            NO RECOMENDADO
          </span>
        )}
      </button>

      {showTooltip && store.warning && (
        <div style={{
          position: 'absolute',
          bottom: 'calc(100% + 8px)',
          left: '50%',
          transform: 'translateX(-50%)',
          background: '#1a0a0a',
          border: '1px solid #ff444466',
          borderRadius: '8px',
          padding: '10px 12px',
          fontSize: '12px',
          color: '#ffaaaa',
          width: '220px',
          textAlign: 'center',
          lineHeight: 1.5,
          zIndex: 10,
          pointerEvents: 'none',
        }}>
          {store.warning}
        </div>
      )}
    </div>
  )
}

// necesitamos React en scope para useState
import React from 'react'