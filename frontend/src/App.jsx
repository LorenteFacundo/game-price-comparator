import { useCallback, useEffect, useRef, useState } from 'react'
import { getDeals, searchGames } from './api/client'
import DealCard from './components/DealCard'
import Footer from './components/Footer'
import GameCard from './components/GameCard'
import SearchPanel from './components/SearchPanel'
import { useLocalStorage } from './hooks/useLocalStorage'

const emptySearch = { results: [], usdRate: 0, warnings: [] }

function savedSearches(items, query) {
  return [query, ...items.filter((item) => item.toLowerCase() !== query.toLowerCase())].slice(0, 6)
}

export default function App() {
  const initialQuery = new URLSearchParams(window.location.search).get('q') || ''
  const [query, setQuery] = useState(initialQuery)
  const [search, setSearch] = useState(emptySearch)
  const [loading, setLoading] = useState(false)
  const [deals, setDeals] = useState([])
  const [dealsLoading, setDealsLoading] = useState(true)
  const [error, setError] = useState('')
  const [currency, setCurrency] = useLocalStorage('pricepulse-currency', 'ARS')
  const [steamMode, setSteamMode] = useLocalStorage('pricepulse-steam-mode', 'regional')
  const [favorites, setFavorites] = useLocalStorage('pricepulse-favorites', [])
  const [history, setHistory] = useLocalStorage('pricepulse-history', [])
  const requestRef = useRef(null)
  const restoredSearchRef = useRef(false)

  const runSearch = useCallback(async (nextQuery, mode = steamMode) => {
    const cleanQuery = nextQuery.trim()
    if (!cleanQuery) return
    requestRef.current?.abort()
    const controller = new AbortController()
    requestRef.current = controller
    setLoading(true)
    setError('')
    setQuery(cleanQuery)
    const params = new URLSearchParams(window.location.search)
    params.set('q', cleanQuery)
    params.set('steam', mode)
    window.history.replaceState({}, '', `${window.location.pathname}?${params.toString()}`)
    try {
      const data = await searchGames(cleanQuery, mode, controller.signal)
      setSearch({ results: data.results || [], usdRate: data.usd_rate || 0, warnings: data.warnings || [] })
      setHistory((items) => savedSearches(items, cleanQuery))
    } catch (requestError) {
      if (requestError.name !== 'CanceledError') {
        setError(requestError.message)
        setSearch(emptySearch)
      }
    } finally {
      if (requestRef.current === controller) setLoading(false)
    }
  }, [setHistory, steamMode])

  useEffect(() => {
    const controller = new AbortController()
    getDeals(controller.signal).then(setDeals).catch(() => setDeals([])).finally(() => setDealsLoading(false))
    return () => controller.abort()
  }, [])

  useEffect(() => {
    if (restoredSearchRef.current) return undefined
    restoredSearchRef.current = true
    const requestedMode = new URLSearchParams(window.location.search).get('steam')
    const initialMode = requestedMode === 'regional' || requestedMode === 'global' ? requestedMode : steamMode
    if (initialMode !== steamMode) setSteamMode(initialMode)
    const startupSearch = initialQuery
      ? window.setTimeout(() => runSearch(initialQuery, initialMode), 0)
      : null
    // The timer intentionally survives React Strict Mode's development remount,
    // so a shared URL is restored exactly once.
    void startupSearch
    return undefined
  }, [initialQuery, runSearch, setSteamMode, steamMode])

  function toggleFavorite(game) {
    setFavorites((items) => items.some((item) => item.id === game.id) ? items.filter((item) => item.id !== game.id) : [game, ...items].slice(0, 24))
  }

  function changeSteamMode(mode) {
    setSteamMode(mode)
    if (search.results.length && query) runSearch(query, mode)
  }

  const favoriteIds = new Set(favorites.map((game) => game.id))

  return (
    <div className="app-shell">
      <a className="skip-link" href="#content">Saltar al contenido</a>
      <header className="topbar">
        <a className="brand" href="/" aria-label="PricePulse, inicio"><span aria-hidden="true">◐</span> PricePulse</a>
        <div className="topbar-meta"><span>Precios reales</span><span aria-hidden="true">·</span><span>ARS primero</span></div>
      </header>

      <main id="content">
        <SearchPanel query={query} onQueryChange={setQuery} onSearch={runSearch} loading={loading} currency={currency} onCurrencyChange={setCurrency} steamMode={steamMode} onSteamModeChange={changeSteamMode} />

        {history.length > 0 && !search.results.length && <section className="recent-searches" aria-label="Búsquedas recientes"><span>Volver a buscar:</span>{history.map((item) => <button type="button" key={item} onClick={() => runSearch(item)}>{item}</button>)}</section>}

        <section className="trust-strip" aria-label="Cómo mostramos las ofertas">
          <span><b>01</b> Moneda original</span><span><b>02</b> Tiendas verificadas</span><span><b>03</b> Sin cuentas compartidas</span>
        </section>

        {error && <div className="notice notice-error" role="alert"><strong>No salió como esperábamos.</strong><span>{error}</span><button type="button" onClick={() => runSearch(query)}>Reintentar</button></div>}
        {search.warnings.map((warning) => <div className="notice" role="status" aria-live="polite" key={warning}>{warning}</div>)}

        {!search.results.length && !loading && <section className="deals-section" aria-labelledby="deals-title">
          <div className="section-heading"><div><span className="eyebrow">Señal de hoy</span><h2 id="deals-title">Ofertas que vale la pena mirar.</h2></div><span className="heading-note">Datos actualizados al consultar</span></div>
          {dealsLoading ? <div className="deal-grid" aria-label="Cargando ofertas">{Array.from({ length: 4 }, (_, index) => <div className="deal-skeleton" key={index} />)}</div> : deals.length ? <div className="deal-grid">{deals.map((deal) => <DealCard key={deal.id} deal={deal} onSearch={runSearch} />)}</div> : <div className="empty-panel"><span aria-hidden="true">⌁</span><h2>Las ofertas vuelven enseguida.</h2><p>Mientras tanto, buscá cualquier juego para comparar sus tiendas.</p></div>}
        </section>}

        {loading && <div className="search-loading" role="status" aria-live="polite"><span className="pulse-dot" /><span>Revisando tiendas y precios…</span></div>}

        {search.results.length > 0 && !loading && <section className="results-section" aria-labelledby="results-title">
          <div className="section-heading"><div><span className="eyebrow">Resultado de búsqueda</span><h2 id="results-title">Coincidencias para “{query}”</h2></div><span className="heading-note">{search.results.length} juego{search.results.length !== 1 ? 's' : ''}</span></div>
          <div className="results-grid">{search.results.map((game) => <GameCard key={game.id} game={game} currency={currency} usdRate={search.usdRate} isFavorite={favoriteIds.has(game.id)} onToggleFavorite={toggleFavorite} />)}</div>
        </section>}

        {favorites.length > 0 && !search.results.length && <section className="favorites-section" aria-labelledby="favorites-title"><div className="section-heading"><div><span className="eyebrow">Tu radar</span><h2 id="favorites-title">Favoritos guardados</h2></div><span className="heading-note">en este dispositivo</span></div><div className="favorite-list">{favorites.slice(0, 6).map((game) => <button key={game.id} type="button" onClick={() => runSearch(game.title)}><span>★</span>{game.title}<i aria-hidden="true">↗</i></button>)}</div></section>}
      </main>
      <Footer />
    </div>
  )
}
