import { useCallback, useEffect, useRef, useState } from 'react'
import { getDeals, getDiscover, getHealth, searchGames } from './api/client'
import Footer from './components/Footer'
import GameCard from './components/GameCard'
import SearchPanel from './components/SearchPanel'
import RankedGameCard from './components/RankedGameCard'
import OffersSection from './components/OffersSection'
import SupportDialog from './components/SupportDialog'
import { useLocalStorage } from './hooks/useLocalStorage'

const emptySearch = { results: [], usdRate: 0, officialRate: 0, cardRate: 0, taxRate: 0, ratesUpdatedAt: '', ratesStale: false, updatedAt: '', warnings: [] }
const emptyDeals = { featured: [], free: [], discounts: [], officialRate: 0, cardRate: 0, taxRate: 0, ratesUpdatedAt: '', ratesStale: false, updatedAt: '', warnings: [] }
const emptyDiscover = { popular: [], mostPlayed: [], updatedAt: '', warnings: [] }
const refreshInterval = 5 * 60 * 1000
const wakeUpNoticeDelay = 4000
const maxStartupRetries = 2
const startupRetryDelay = 3000

function savedSearches(items, query) {
  return [query, ...items.filter((item) => item.toLowerCase() !== query.toLowerCase())].slice(0, 6)
}

export default function App() {
  const initialQuery = new URLSearchParams(window.location.search).get('q') || ''
  const [query, setQuery] = useState(initialQuery)
  const [search, setSearch] = useState(emptySearch)
  const [hasSearched, setHasSearched] = useState(Boolean(initialQuery))
  const [loading, setLoading] = useState(false)
  const [deals, setDeals] = useState(emptyDeals)
  const [dealsLoading, setDealsLoading] = useState(true)
  const [discover, setDiscover] = useState(emptyDiscover)
  const [discoverLoading, setDiscoverLoading] = useState(true)
  const [error, setError] = useState('')
  const [currency, setCurrency] = useLocalStorage('pricepulse-currency', 'ARS')
  const [favorites, setFavorites] = useLocalStorage('pricepulse-favorites', [])
  const [history, setHistory] = useLocalStorage('pricepulse-history', [])
  const [health, setHealth] = useState({ state: 'checking', data: null, message: '' })
  const [support, setSupport] = useState({ open: false, view: 'feedback', context: null })
  const requestRef = useRef(null)
  const restoredSearchRef = useRef(false)
  const homepageRefreshRef = useRef(0)
  const [slowRequest, setSlowRequest] = useState(false)
  const waitingForAPI = loading || dealsLoading || discoverLoading

  useEffect(() => {
    if (!waitingForAPI) return undefined
    const timer = window.setTimeout(() => setSlowRequest(true), wakeUpNoticeDelay)
    return () => { window.clearTimeout(timer); setSlowRequest(false) }
  }, [waitingForAPI])

  const runSearch = useCallback(async (nextQuery) => {
    const cleanQuery = nextQuery.trim()
    if (!cleanQuery) return
    requestRef.current?.abort()
    const controller = new AbortController()
    requestRef.current = controller
    setLoading(true)
    setHasSearched(true)
    setError('')
    setQuery(cleanQuery)
    const params = new URLSearchParams(window.location.search)
    params.set('q', cleanQuery)
    params.delete('steam')
    window.history.replaceState({}, '', `${window.location.pathname}?${params.toString()}`)
    try {
      const data = await searchGames(cleanQuery, controller.signal)
      setSearch({ results: data.results || [], usdRate: data.usd_rate || 0, officialRate: data.official_rate || 0, cardRate: data.card_rate || 0, taxRate: data.tax_rate || 0, ratesUpdatedAt: data.rates_updated_at || '', ratesStale: Boolean(data.rates_stale), updatedAt: data.updated_at || '', warnings: data.warnings || [] })
      setHistory((items) => savedSearches(items, cleanQuery))
    } catch (requestError) {
      if (requestError.name !== 'CanceledError') {
        setError(requestError.message)
        setSearch(emptySearch)
      }
    } finally {
      if (requestRef.current === controller) setLoading(false)
    }
  }, [setHistory])

  useEffect(() => {
    const controller = new AbortController()
    let retryTimer = 0
    async function refreshHomepage(force = false, attempt = 0) {
      if (document.hidden || (!force && Date.now() - homepageRefreshRef.current < refreshInterval)) return
      homepageRefreshRef.current = Date.now()
      const [dealsResult, discoverResult] = await Promise.allSettled([getDeals(controller.signal), getDiscover(controller.signal)])
      if (controller.signal.aborted) return
      // El backend gratuito se duerme sin tráfico y puede tardar ~30-50 s en
      // despertar. En la carga inicial reintentamos antes de mostrar un error.
      const bothFailed = [dealsResult, discoverResult].every((result) => result.status === 'rejected' && result.reason?.name !== 'CanceledError')
      if (bothFailed && attempt < maxStartupRetries && !controller.signal.aborted) {
        retryTimer = window.setTimeout(() => refreshHomepage(true, attempt + 1), startupRetryDelay)
        return
      }
      if (dealsResult.status === 'fulfilled') setDeals(dealsResult.value)
      else if (dealsResult.reason?.name !== 'CanceledError') setDeals((current) => ({ ...current, warnings: ['No pudimos actualizar las ofertas.'] }))
      if (discoverResult.status === 'fulfilled') setDiscover(discoverResult.value)
      else if (discoverResult.reason?.name !== 'CanceledError') setDiscover((current) => ({ ...current, warnings: ['No pudimos actualizar los rankings.'] }))
      setDealsLoading(false)
      setDiscoverLoading(false)
    }
    refreshHomepage(true)
    const interval = window.setInterval(() => refreshHomepage(), refreshInterval)
    const onVisibilityChange = () => { if (!document.hidden) refreshHomepage() }
    document.addEventListener('visibilitychange', onVisibilityChange)
    return () => { controller.abort(); window.clearTimeout(retryTimer); window.clearInterval(interval); document.removeEventListener('visibilitychange', onVisibilityChange) }
  }, [])

  useEffect(() => {
    let active = true
    async function checkHealth() {
      try {
        const data = await getHealth()
        if (active) setHealth({ state: 'online', data, message: '' })
      } catch (healthError) {
        if (active) setHealth({ state: 'offline', data: null, message: healthError.message })
      }
    }
    checkHealth()
    const interval = window.setInterval(checkHealth, 60000)
    return () => { active = false; window.clearInterval(interval) }
  }, [])

  useEffect(() => {
    if (restoredSearchRef.current) return undefined
    restoredSearchRef.current = true
    const startupSearch = initialQuery
      ? window.setTimeout(() => runSearch(initialQuery), 0)
      : null
    // The timer intentionally survives React Strict Mode's development remount,
    // so a shared URL is restored exactly once.
    void startupSearch
    return undefined
  }, [initialQuery, runSearch])

  function toggleFavorite(game) {
    setFavorites((items) => items.some((item) => item.id === game.id) ? items.filter((item) => item.id !== game.id) : [game, ...items].slice(0, 24))
  }

  const favoriteIds = new Set(favorites.map((game) => game.id))
  const taxPercent = Math.round(search.taxRate * 100)
  const formattedCardRate = search.cardRate > 0 ? formatRate(search.cardRate) : 'no disponible'
  const formattedOfficialRate = search.officialRate > 0 ? formatRate(search.officialRate) : 'no disponible'
  const updates = { search: search.updatedAt, deals: deals.updatedAt, discover: discover.updatedAt, rates: search.ratesUpdatedAt || deals.ratesUpdatedAt }
  const sourceWarnings = [...new Set([...(search.warnings || []), ...(deals.warnings || []), ...(discover.warnings || [])])]
  const openSupport = useCallback((view = 'feedback', context = null) => setSupport({ open: true, view, context }), [])
  const closeSupport = useCallback(() => setSupport((current) => ({ ...current, open: false })), [])

  return (
    <div className="app-shell">
      <a className="skip-link" href="#content">Saltar al contenido</a>
      <header className="topbar">
        <a className="brand" href="/" aria-label="PricePulse, inicio"><span aria-hidden="true">◐</span> PricePulse</a>
        <div className="topbar-meta">Un juego · varias tiendas · el menor precio</div>
      </header>

      <main id="content">
        <SearchPanel query={query} onQueryChange={setQuery} onSearch={runSearch} loading={loading} currency={currency} onCurrencyChange={setCurrency} />

        {history.length > 0 && !hasSearched && <section className="recent-searches" aria-label="Búsquedas recientes"><span>Volver a buscar:</span>{history.map((item) => <button type="button" key={item} onClick={() => runSearch(item)}>{item}</button>)}</section>}

        {waitingForAPI && slowRequest && <div className="notice notice-waking" role="status" aria-live="polite"><span className="pulse-dot" /><span>Estamos despertando el servidor (corre en un plan gratuito). La primera carga puede tardar hasta un minuto; después es instantánea.</span></div>}
        {error && <div className="notice notice-error" role="alert"><span>{error}</span><button type="button" onClick={() => runSearch(query)}>Reintentar</button></div>}
        {search.warnings.map((warning) => <div className="notice" role="status" aria-live="polite" key={warning}>{warning}</div>)}

        {!hasSearched && !loading && <>
          <section className="rank-section" aria-labelledby="popular-title">
            <div className="section-heading"><h2 id="popular-title">Populares en Steam</h2></div>
            {discoverLoading ? <div className="rank-grid" aria-label="Cargando populares">{Array.from({ length: 4 }, (_, index) => <div className="deal-skeleton" key={index} />)}</div> : discover.popular.length > 0 && <div className="rank-grid">{discover.popular.map((game) => <RankedGameCard key={game.id} game={game} onSearch={runSearch} />)}</div>}
          </section>

          <section className="rank-section" aria-labelledby="most-played-title">
            <div className="section-heading"><h2 id="most-played-title">Más jugados en Steam</h2></div>
            {discoverLoading ? <div className="rank-grid" aria-label="Cargando más jugados">{Array.from({ length: 4 }, (_, index) => <div className="deal-skeleton" key={index} />)}</div> : discover.mostPlayed.length > 0 && <div className="rank-grid">{discover.mostPlayed.map((game) => <RankedGameCard key={game.id} game={game} onSearch={runSearch} showPlayers />)}</div>}
          </section>

          <OffersSection deals={deals} loading={dealsLoading} currency={currency} onSearch={runSearch} onReport={(context) => openSupport('feedback', context)} />
        </>}

        {loading && <div className="search-loading" role="status" aria-live="polite"><span className="pulse-dot" /><span>Buscando precios…</span></div>}
        {hasSearched && !loading && !error && search.results.length === 0 && <section className="empty-panel" aria-live="polite"><h2>No encontramos “{query}”</h2><p>Probá con el nombre completo o revisá cómo está escrito.</p><button type="button" onClick={() => runSearch(query)}>Reintentar</button></section>}

        {search.results.length > 0 && !loading && <section className="results-section" aria-labelledby="results-title">
          <div className="section-heading"><h2 id="results-title">{query}</h2><span className="heading-note">{search.results.length} juego{search.results.length !== 1 ? 's' : ''}</span></div>
          <div className="tax-note">{currency === 'ARS' ? `ARS + IMP · USD × ${formattedCardRate} (oficial ${formattedOfficialRate} + IVA ${taxPercent}%) = total final. ARS regional no se reconvierte.` : 'USD · Sólo precios publicados en USD para Argentina. No se convierte ningún precio.'}</div>
          <div className="results-grid">{search.results.map((game) => <GameCard key={game.id} game={game} currency={currency} officialRate={search.officialRate} cardRate={search.cardRate} taxRate={search.taxRate} isFavorite={favoriteIds.has(game.id)} onToggleFavorite={toggleFavorite} onReport={(context) => openSupport('feedback', context)} />)}</div>
        </section>}

        {favorites.length > 0 && !hasSearched && <section className="favorites-section" aria-labelledby="favorites-title"><div className="section-heading"><h2 id="favorites-title">Favoritos</h2></div><div className="favorite-list">{favorites.slice(0, 6).map((game) => <button key={game.id} type="button" onClick={() => runSearch(game.title)}><span>★</span>{game.title}<i aria-hidden="true">↗</i></button>)}</div></section>}
      </main>
      <Footer health={health} updates={updates} onOpenSupport={openSupport} />
      <SupportDialog open={support.open} initialView={support.view} context={support.context} health={health} updates={updates} warnings={sourceWarnings} onClose={closeSupport} />
    </div>
  )
}

function formatRate(rate) {
  return new Intl.NumberFormat('es-AR', { maximumFractionDigits: 0 }).format(rate) + ' ARS/USD'
}
