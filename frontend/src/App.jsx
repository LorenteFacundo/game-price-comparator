import { useState } from 'react'
import { searchGame } from './api/search'
import SearchBar from './components/SearchBar'
import GameCard from './components/GameCard'
import StoreButtons from './components/StoreButtons'

export default function App() {
  const [results, setResults] = useState(null)
  const [usdRate, setUsdRate] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [showARS, setShowARS] = useState(true)
  const [lastQuery, setLastQuery] = useState('')

  async function handleSearch(query) {
    setLoading(true)
    setError(null)
    setLastQuery(query)
    try {
      const data = await searchGame(query)
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

  return (
    <div style={{ maxWidth: '700px', margin: '0 auto', padding: '2rem 1rem 4rem' }}>

      {/* Header */}
      <div style={{ textAlign: 'center', marginBottom: '2.5rem' }}>
        <h1 style={{
          fontSize: '28px',
          fontWeight: 700,
          background: 'linear-gradient(135deg, #fff 0%, #a78bfa 50%, #60c8ff 100%)',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent',
          marginBottom: '8px',
        }}>
          🎮 Comparador de precios
        </h1>
        <p style={{ color: 'var(--muted)', fontSize: '14px' }}>
          Encontrá el precio más barato entre todas las tiendas
        </p>
      </div>

      {/* Search */}
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px', marginBottom: '2rem' }}>
        <SearchBar onSearch={handleSearch} loading={loading} />

        {/* Toggle ARS/USD */}
        <div style={{
          display: 'flex',
          background: 'var(--surface)',
          border: '1px solid var(--border)',
          borderRadius: '8px',
          padding: '3px',
          gap: '3px',
        }}>
          {['ARS', 'USD'].map(currency => (
            <button
              key={currency}
              onClick={() => setShowARS(currency === 'ARS')}
              style={{
                background: (currency === 'ARS') === showARS ? 'var(--accent)' : 'transparent',
                border: 'none',
                borderRadius: '6px',
                padding: '5px 16px',
                fontSize: '13px',
                fontWeight: 600,
                color: (currency === 'ARS') === showARS ? '#fff' : 'var(--muted)',
                transition: 'all 0.2s',
              }}
            >
              {currency}
            </button>
          ))}
        </div>

        <StoreButtons query={lastQuery} />

        {usdRate > 0 && (
          <p style={{ fontSize: '12px', color: 'var(--muted)' }}>
            💵 Dólar blue: ${usdRate.toLocaleString('es-AR')}
          </p>
        )}
      </div>

      {/* Estados */}
      {loading && (
        <p style={{ textAlign: 'center', color: 'var(--muted)', fontSize: '15px' }}>
          Buscando precios...
        </p>
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
        }}>
          {error}
        </div>
      )}

      {results && results.length === 0 && (
        <p style={{ textAlign: 'center', color: 'var(--muted)' }}>
          No se encontraron resultados para "{lastQuery}"
        </p>
      )}

      {/* Resultados */}
      {results && results.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <p style={{ fontSize: '13px', color: 'var(--muted)' }}>
            {results.length} resultado{results.length !== 1 ? 's' : ''} para "{lastQuery}"
          </p>
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
    </div>
  )
}