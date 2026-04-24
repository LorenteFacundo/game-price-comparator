import { useState } from 'react'
import { searchGame } from './api/search'
import SearchBar from './components/SearchBar'
import GameCard from './components/GameCard'
import StoreButtons from './components/StoreButtons'
import Footer from './components/Footer'

const STEAM_MODES = [
  { value: 'regional', label: 'Steam AR' },
  { value: 'global', label: 'Steam Global' },
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
    if (lastQuery) runSearch(lastQuery, nextMode, false)
  }

  return (
    <div style={{ maxWidth: '700px', margin: '0 auto', padding: '2rem 1rem 4rem' }}>
      <div style={{ textAlign: 'center', marginBottom: '2rem' }}>
        <h1 style={{
          fontSize: '28px',
          fontWeight: 700,
          background: 'linear-gradient(135deg, #fff 0%, #79c7ff 40%, #7c5cfc 100%)',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent',
          marginBottom: '8px',
        }}>
          Comparador de precios
        </h1>
        <p style={{ color: 'var(--muted)', fontSize: '14px' }}>
          Busca un juego y compara precios rapido
        </p>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', marginBottom: '1.5rem' }}>
        <SearchBar onSearch={handleSearch} loading={loading} />

        <div style={{
          display: 'flex',
          justifyContent: 'flex-end',
          gap: '8px',
          flexWrap: 'wrap',
          alignItems: 'center',
        }}>
          <div style={{
            display: 'flex',
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: '999px',
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
                  borderRadius: '999px',
                  padding: '6px 10px',
                  fontSize: '12px',
                  fontWeight: 700,
                  color: (currency === 'ARS') === showARS ? '#fff' : 'var(--muted)',
                }}
              >
                {currency}
              </button>
            ))}
          </div>

          <div style={{
            display: 'flex',
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: '999px',
            padding: '3px',
            gap: '3px',
          }}>
            {STEAM_MODES.map(mode => (
              <button
                key={mode.value}
                onClick={() => handleSteamModeChange(mode.value)}
                style={{
                  background: steamMode === mode.value ? 'var(--accent)' : 'transparent',
                  border: 'none',
                  borderRadius: '999px',
                  padding: '6px 10px',
                  fontSize: '12px',
                  fontWeight: 700,
                  color: steamMode === mode.value ? '#fff' : 'var(--muted)',
                }}
              >
                {mode.label}
              </button>
            ))}
          </div>
        </div>

        <StoreButtons query={lastQuery} />
      </div>

      {loading && (
        <p style={{ textAlign: 'center', color: 'var(--muted)', fontSize: '15px', marginBottom: '1rem' }}>
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
          marginBottom: '1rem',
        }}>
          {error}
        </div>
      )}

      {!loading && results && results.length === 0 && (
        <p style={{ textAlign: 'center', color: 'var(--muted)' }}>
          No se encontraron resultados para "{lastQuery}"
        </p>
      )}

      {!loading && results && results.length > 0 && (
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

      <Footer />
    </div>
  )
}
