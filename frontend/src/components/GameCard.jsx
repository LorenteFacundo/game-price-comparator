import { useState } from 'react'
import { displayFinalMoney } from '../api/client'
import PriceRow from './PriceRow'

export default function GameCard({ game, currency, officialRate, cardRate, taxRate, isFavorite, onToggleFavorite }) {
  const [expanded, setExpanded] = useState(true)
  const best = displayFinalMoney(game.best_deal, currency, officialRate, cardRate, taxRate)
  const priceCount = game.prices?.filter((price) => price.price > 0).length || 0
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
        {game.best_deal ? <div className="best-summary"><strong>{best.label}</strong><small>{best.note}</small></div> : <p className="muted-copy">Sin precios comparables.</p>}
        {game.prices?.length > 0 && <button className="show-prices" type="button" aria-expanded={expanded} onClick={() => setExpanded((value) => !value)}>{expanded ? 'Ocultar' : `Ver ${priceCount} tiendas`} <span aria-hidden="true">{expanded ? '−' : '+'}</span></button>}
      </div>
      {expanded && game.prices?.length > 0 && <ul className="price-list">{game.prices.map((price) => <PriceRow key={`${price.store_name}-${price.url}`} price={price} preferredCurrency={currency} officialRate={officialRate} cardRate={cardRate} taxRate={taxRate} isBest={game.best_deal?.store_name === price.store_name && game.best_deal?.url === price.url} />)}</ul>}
    </article>
  )
}
