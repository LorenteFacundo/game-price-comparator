import { useMemo, useState } from 'react'
import DealCard from './DealCard'

const tabs = [
  { id: 'featured', label: 'Destacadas' },
  { id: 'free', label: 'Gratis ahora' },
  { id: 'discounts', label: 'Más descuento' },
]

const stores = [
  { id: 'all', label: 'Todas' },
  { id: 'steam', label: 'Steam' },
  { id: 'epic', label: 'Epic' },
  { id: 'microsoft', label: 'Microsoft' },
]

function storeKey(name = '') {
  const normalized = name.toLowerCase()
  if (normalized.includes('steam')) return 'steam'
  if (normalized.includes('epic')) return 'epic'
  if (normalized.includes('microsoft')) return 'microsoft'
  return 'other'
}

function canonicalTitle(title = '') {
	return title.toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/[^a-z0-9]+/g, ' ').replace(/\b(deluxe|ultimate|complete|gold|collector s|game of the year) edition\b/g, '').trim()
}

export default function OffersSection({ deals, loading, currency, onSearch, onReport }) {
  const [activeTab, setActiveTab] = useState('featured')
  const [activeStore, setActiveStore] = useState('all')
  const visibleDeals = useMemo(() => {
	const collection = (deals[activeTab] || []).filter((deal) => ['ARS', 'USD'].includes(deal.currency?.toUpperCase()) && (currency !== 'USD' || deal.currency.toUpperCase() === 'USD'))
	const byStore = activeStore === 'all' ? collection : collection.filter((deal) => storeKey(deal.store_name) === activeStore)
	const seen = new Set()
	return byStore.filter((deal) => {
	  const key = canonicalTitle(deal.title)
	  if (seen.has(key)) return false
	  seen.add(key)
	  return true
	}).slice(0, 12)
	}, [activeStore, activeTab, currency, deals])

  return (
    <section className="deals-section" aria-labelledby="deals-title">
      <div className="offers-heading">
        <h2 id="deals-title">Ofertas</h2>
        <div className="offer-tabs" role="tablist" aria-label="Tipo de oferta">
          {tabs.map((tab) => <button key={tab.id} type="button" role="tab" aria-selected={activeTab === tab.id} className={activeTab === tab.id ? 'is-active' : ''} onClick={() => setActiveTab(tab.id)}>{tab.label}<span>{deals[tab.id]?.length || 0}</span></button>)}
        </div>
      </div>
	  <div className="store-filters" aria-label="Filtrar por tienda">
        {stores.map((store) => <button key={store.id} type="button" className={activeStore === store.id ? 'is-active' : ''} aria-pressed={activeStore === store.id} onClick={() => setActiveStore(store.id)}>{store.label}</button>)}
	  </div>
	  <div className="tax-note offer-tax-note">{currency === 'ARS' ? (deals.cardRate > 0 ? `ARS + IMP · USD × ${new Intl.NumberFormat('es-AR', { maximumFractionDigits: 0 }).format(deals.cardRate)} con IVA ${Math.round(deals.taxRate * 100)}%. Los precios ARS no se reconvierten.` : 'ARS + IMP · Cotización no disponible; los precios USD se muestran sin convertir.') : 'USD · Sólo ofertas publicadas en dólares para Argentina.'}</div>
      {deals.warnings?.map((warning) => <div className="notice" role="status" key={warning}>{warning}</div>)}
	  {loading ? <div className="deal-grid" aria-label="Cargando ofertas">{Array.from({ length: 4 }, (_, index) => <div className="deal-skeleton" key={index} />)}</div> : visibleDeals.length > 0 ? <div className="deal-grid" role="tabpanel">{visibleDeals.map((deal, index) => <DealCard key={`${deal.id}-${deal.store_name}`} deal={deal} currency={currency} officialRate={deals.officialRate} cardRate={deals.cardRate} taxRate={deals.taxRate} onSearch={onSearch} onReport={onReport} lead={activeTab === 'featured' && index === 0} />)}</div> : <div className="empty-panel compact-empty">No hay ofertas para este filtro.</div>}
    </section>
  )
}
