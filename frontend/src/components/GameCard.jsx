import { useState } from 'react'
import { displayFinalMoney } from '../api/client'
import PriceRow from './PriceRow'

export default function GameCard({ game, currency, officialRate, cardRate, taxRate, isFavorite, onToggleFavorite }) {
  const [expanded, setExpanded] = useState(true)
  const visiblePrices = (game.prices || []).filter((price) => price.price > 0 && (currency !== 'USD' || price.currency === 'USD'))
  const bestDeal = currency === 'USD'
    ? [...visiblePrices].sort((left, right) => left.price - right.price)[0]
    : game.best_deal
  const best = displayFinalMoney(bestDeal, currency, officialRate, cardRate, taxRate)
  return (
    <article className="game-card">
      <div className="game-cover">
        {game.image_url ? <img src={game.image_url} alt="" width="300" height="140" loading="lazy" /> : <div className="image-fallback" aria-hidden="true">◌</div>}
      </div>
      <div className="game-main">
        <div className="game-heading">
          <h3>{game.title}</h3>
          <button className={`favorite-button ${isFavorite ? 'is-saved' : ''}`} type="button" aria-label={isFavorite ? `Quitar ${game.title} de favoritos` : `Guardar ${game.title} en favoritos`} aria-pressed={isFavorite} onClick={() => onToggleFavorite(game)}>{isFavorite ? '★' : '☆'}</button>
        </div>
        {bestDeal ? <div className="best-summary"><strong>{best.label}</strong><small>{best.note}</small></div> : <p className="muted-copy">Sin precio publicado en USD.</p>}
        {visiblePrices.length > 0 && <button className="show-prices" type="button" aria-expanded={expanded} onClick={() => setExpanded((value) => !value)}>{expanded ? 'Ocultar' : `Ver ${visiblePrices.length} tiendas`} <span aria-hidden="true">{expanded ? '−' : '+'}</span></button>}
      </div>
      {expanded && visiblePrices.length > 0 && <ul className="price-list">{visiblePrices.map((price) => <PriceRow key={`${price.store_name}-${price.url}`} price={price} preferredCurrency={currency} officialRate={officialRate} cardRate={cardRate} taxRate={taxRate} isBest={bestDeal?.store_name === price.store_name && bestDeal?.url === price.url} />)}</ul>}
    </article>
  )
}
