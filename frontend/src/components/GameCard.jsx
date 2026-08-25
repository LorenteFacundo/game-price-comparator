import { useState } from 'react'
import { displayMoney } from '../api/client'
import PriceRow from './PriceRow'

export default function GameCard({ game, currency, usdRate, isFavorite, onToggleFavorite }) {
  const [expanded, setExpanded] = useState(true)
  const best = displayMoney(game.best_deal, currency, usdRate)
  const priceCount = game.prices?.filter((price) => price.price > 0).length || 0
  return (
    <article className="game-card">
      <div className="game-cover">
        {game.image_url ? <img src={game.image_url} alt="" width="300" height="140" loading="lazy" /> : <div className="image-fallback" aria-hidden="true">◌</div>}
      </div>
      <div className="game-main">
        <div className="game-heading">
          <div>
            <span className="eyebrow">{priceCount ? `${priceCount} precios comparables` : 'Sin precios comparables'}</span>
            <h3>{game.title}</h3>
          </div>
          <button className={`favorite-button ${isFavorite ? 'is-saved' : ''}`} type="button" aria-label={isFavorite ? `Quitar ${game.title} de favoritos` : `Guardar ${game.title} en favoritos`} aria-pressed={isFavorite} onClick={() => onToggleFavorite(game)}>{isFavorite ? '★' : '☆'}</button>
        </div>
        {game.best_deal ? <div className="best-summary"><span>Desde</span><strong>{best.label}</strong>{best.converted && <small>conversión estimada</small>}</div> : <p className="muted-copy">Todavía no hay una oferta comparable para este juego.</p>}
        {game.prices?.length > 0 && <button className="show-prices" type="button" aria-expanded={expanded} onClick={() => setExpanded((value) => !value)}>{expanded ? 'Ocultar tiendas' : `Ver ${game.prices.length} tiendas`} <span aria-hidden="true">{expanded ? '−' : '+'}</span></button>}
      </div>
      {expanded && game.prices?.length > 0 && <ul className="price-list">{game.prices.map((price) => <PriceRow key={`${price.store_name}-${price.url}`} price={price} preferredCurrency={currency} usdRate={usdRate} isBest={game.best_deal?.store_name === price.store_name && game.best_deal?.url === price.url} />)}</ul>}
    </article>
  )
}
