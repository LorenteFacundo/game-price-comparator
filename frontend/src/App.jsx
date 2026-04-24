import { useState } from 'react'
import { searchGame } from './api/search'
import SearchBar from './components/SearchBar'
import GameCard from './components/GameCard'
import StoreButtons from './components/StoreButtons'
import Footer from './components/Footer'

const STEAM_MODES = [
  {
    value: 'regional',
    label: 'Steam AR / LATAM-USD',
    help: 'Usa el precio mostrado por Steam para Argentina. Puede venir en ARS o en USD segun el juego.',
  },
  {
    value: 'global',
    label: 'Steam Global / US',
    help: 'Usa el precio de Steam en Estados Unidos para comparar contra el valor internacional.',
  },
]

export default function App() {
  const [results, setResults] = useState(null)
  const [usdRate, setUsdRate] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [showARS, setShowARS] = useState(true)
  const [lastQuery, setLastQuery] = useState('')
  const [steamMode, setSteamMode] = useState('regional')

  async function runSearch(query, mode, updateQuery = true) {
    setLoading(true)
    setError(null)
    if (updateQuery) setLastQuery(query)

    try {
      const data = await searchGame(query, mode)
      if (data.error) throw new Error(data.error)
      setResults(data.results)
      setUsdRate(data.usd_rate)
    } catch (e) {
      setError(e.message)
      setResults(null)
    } finally {
      setLoading(false)
    }
  }

  function handleSearch(query) {
    runSearch(query, steamMode, true)
  }

  function handleSteamModeChange(nextMode) {
    setSteamMode(nextMode)
    if (lastQuery) {
      runSearch(lastQuery, nextMode, false)
    }
  }

  const currentSteamMode = STEAM_MODES.find(mode => mode.value === steamMode)

  return (
    <div style={{ maxWidth: '760px', margin: '0 auto', padding: '2rem 1rem 4rem' }}>
      <div style={{ textAlign: 'center', marginBottom: '2rem' }}>
        <h1 style={{
          fontSize: '32px',
          fontWeight: 800,
          background: 'linear-gradient(135deg, #fff 0%, #79c7ff 35%, #7c5cfc 100%)',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent',
          marginBottom: '10px',
        }}>
          Comparador de precios
        </h1>
        <p style={{ color: 'var(--muted)', fontSize: '15px', maxWidth: '580px', margin: '0 auto' }}>
          Busca un juego y compara rapido entre Steam y otras tiendas con foco en precios para Argentina.
        </p>
      </div>

      <section style={{
        background: 'linear-gradient(180deg, rgba(255,255,255,0.03), rgba(255,255,255,0.015))',
        border: '1px solid var(--border)',
        borderRadius: '18px',
        padding: '18px',
        display: 'flex',
        flexDirection: 'column',
        gap: '16px',
        marginBottom: '1.5rem',
      }}>
        <SearchBar onSearch={handleSearch} loading={loading} />

        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
          gap: '12px',
          width: '100%',
        }}>
          <div style={{
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: '14px',
            padding: '12px',
          }}>
            <p style={{ fontSize: '12px', color: 'var(--muted)', marginBottom: '10px' }}>
              Moneda mostrada
            </p>
            <div style={{
              display: 'flex',
              background: 'var(--surface2)',
              border: '1px solid var(--border)',
              borderRadius: '10px',
              padding: '3px',
              gap: '3px',
            }}>
              {['ARS', 'USD'].map(currency => (
                <button
                  key={currency}
                  onClick={() => setShowARS(currency === 'ARS')}
                  style={{
                    flex: 1,
                    background: (currency === 'ARS') === showARS ? 'var(--accent)' : 'transparent',
                    border: 'none',
                    borderRadius: '8px',
                    padding: '8px 12px',
                    fontSize: '13px',
                    fontWeight: 700,
                    color: (currency === 'ARS') === showARS ? '#fff' : 'var(--muted)',
                    transition: 'all 0.2s',
                  }}
                >
                  {currency}
                </button>
              ))}
            </div>
          </div>

          <div style={{
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: '14px',
            padding: '12px',
          }}>
            <p style={{ fontSize: '12px', color: 'var(--muted)', marginBottom: '10px' }}>
              Precio de Steam
            </p>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {STEAM_MODES.map(mode => (
                <button
                  key={mode.value}
                  onClick={() => handleSteamModeChange(mode.value)}
                  style={{
                    textAlign: 'left',
                    background: steamMode === mode.value ? 'rgba(124,92,252,0.16)' : 'var(--surface2)',
                    border: `1px solid ${steamMode === mode.value ? 'rgba(124,92,252,0.5)' : 'var(--border)'}`,
                    borderRadius: '10px',
                    padding: '10px 12px',
                    color: 'var(--text)',
                  }}
                >
                  <div style={{ fontSize: '13px', fontWeight: 700, marginBottom: '4px' }}>
                    {mode.label}
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--muted)', lineHeight: 1.4 }}>
                    {mode.help}
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>

        <div style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: '8px',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}>
          <StoreButtons query={lastQuery} />
          {usdRate > 0 && (
            <div style={{
              fontSize: '12px',
              color: 'var(--muted)',
              background: 'var(--surface)',
              border: '1px solid var(--border)',
              borderRadius: '999px',
              padding: '8px 12px',
            }}>
              Dolar blue: ${usdRate.toLocaleString('es-AR')}
            </div>
          )}
        </div>

        <div style={{
          background: 'rgba(79,187,255,0.08)',
          border: '1px solid rgba(79,187,255,0.2)',
          borderRadius: '12px',
          padding: '12px 14px',
          fontSize: '12px',
          lineHeight: 1.5,
          color: '#b7dcff',
        }}>
          Steam usa por defecto <strong>{currentSteamMode?.label}</strong>. Si cambias esta opcion y ya habia una busqueda hecha,
          los resultados se actualizan automaticamente. En algunos juegos, Steam para Argentina muestra precios en USD en lugar de ARS.
        </div>
      </section>

      {loading && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '1rem' }}>
          {[1, 2, 3].map(item => (
            <div
              key={item}
              style={{
                background: 'var(--surface)',
                border: '1px solid var(--border)',
                borderRadius: '16px',
                height: '104px',
                opacity: 0.7,
              }}
            />
          ))}
        </div>
      )}

      {error && (
        <div style={{
          background: 'rgba(255,107,107,0.1)',
          border: '1px solid rgba(255,107,107,0.3)',
          borderRadius: 'var(--radius)',
          padding: '14px',
          color: 'var(--red)',
          fontSize: '14px',
          textAlign: 'center',
          marginBottom: '1rem',
        }}>
          {error}
        </div>
      )}

      {!loading && results && results.length === 0 && (
        <div style={{
          background: 'var(--surface)',
          border: '1px solid var(--border)',
          borderRadius: '16px',
          padding: '18px',
          textAlign: 'center',
          color: 'var(--muted)',
        }}>
          No se encontraron resultados para "{lastQuery}". Prueba con el nombre original del juego o una busqueda mas corta.
        </div>
      )}

      {!loading && results && results.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            gap: '12px',
            flexWrap: 'wrap',
            alignItems: 'center',
          }}>
            <p style={{ fontSize: '13px', color: 'var(--muted)' }}>
              {results.length} resultado{results.length !== 1 ? 's' : ''} para "{lastQuery}"
            </p>
            <p style={{ fontSize: '13px', color: 'var(--muted)' }}>
              Steam activo: {currentSteamMode?.label}
            </p>
          </div>

          {results.map(game => (
            <GameCard
              key={game.id}
              game={game}
              usdRate={usdRate}
              showARS={showARS}
            />
          ))}
        </div>
      )}

      <Footer />
    </div>
  )
}
