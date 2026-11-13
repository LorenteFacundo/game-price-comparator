import { displayFinalMoney } from '../api/client'
import GameImage from './GameImage'

function playerCount(players) {
  if (!players) return ''
  return new Intl.NumberFormat('es-AR', { notation: 'compact', maximumFractionDigits: 1 }).format(players)
}

function samePrice(left, right) {
  return left && right && left.store_name === right.store_name && left.price === right.price && left.currency === right.currency
}

export default function RankedGameCard({ game, onSearch, currency, officialRate, cardRate, taxRate, showPlayers = false }) {
  const players = playerCount(game.players)
  const money = (price) => displayFinalMoney(price, currency, officialRate, cardRate, taxRate).label
  const steam = game.steam_price
  // En modo USD sólo mostramos precios publicados en dólares.
  const best = game.best_price && (currency !== 'USD' || game.best_price.currency === 'USD') ? game.best_price : null
  const steamIsBest = samePrice(steam, best)
  const hasPrices = Boolean(steam || best)

  return (
    <article className="rank-card">
      <button type="button" onClick={() => onSearch(game.title)} aria-label={`Comparar precios de ${game.title}`}>
        <div className="rank-image">
          <GameImage src={game.image_url} />
          <span className="rank-number">#{game.rank}</span>
        </div>
        <span className="rank-title">{game.title}</span>
        {hasPrices && <dl className="rank-prices">
          {steam && <div className={steamIsBest ? 'is-best' : ''}><dt>Steam</dt><dd>{money(steam)}</dd></div>}
          {best && !steamIsBest && <div className="is-best"><dt>Más barato · {best.store_name}</dt><dd>{money(best)}</dd></div>}
          {steamIsBest && <div className="rank-best-note"><dt>Steam es la opción más barata</dt></div>}
        </dl>}
        <span className="rank-action">{showPlayers && players ? `${players} pico reciente · ` : ''}{hasPrices ? 'Ver todas las tiendas' : 'Comparar precios'} <b aria-hidden="true">↗</b></span>
      </button>
    </article>
  )
}
