import PriceRow from './PriceRow'
import { formatARS, formatUSD, toARS } from '../api/search'

export default function GameCard({ game, usdRate, showARS }) {
  if (!game.prices || game.prices.length === 0) {
    return (
      <div style={{
        background: 'var(--surface)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius)',
        padding: '20px',
        opacity: 0.6,
      }}>
        <p style={{ fontSize: '16px', fontWeight: 600 }}>{game.title}</p>
        <p style={{ fontSize: '13px', color: 'var(--muted)', marginTop: '6px' }}>
          Sin precios disponibles en este momento
        </p>
      </div>
    )
  }

const bestPrice = showARS
  ? formatARS(toARS(game.best_deal?.price_usd, usdRate))
  : formatUSD(game.best_deal?.price_usd)

  return (
    <div style={{
      background: 'var(--surface)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius)',
      overflow: 'hidden',
    }}>
      <div style={{ display: 'flex', gap: '16px', padding: '16px' }}>
        {game.image_url && (
          <img
            src={game.image_url}
            alt={game.title}
            style={{
              width: '120px',
              height: '56px',
              objectFit: 'cover',
              borderRadius: '8px',
              flexShrink: 0,
            }}
          />
        )}
        <div style={{ flex: 1, minWidth: 0 }}>
          <h3 style={{ fontSize: '16px', fontWeight: 600, marginBottom: '4px' }}>
            {game.title}
          </h3>
          <p style={{ fontSize: '13px', color: 'var(--muted)' }}>
            Desde {bestPrice} · {game.prices.length} tienda{game.prices.length !== 1 ? 's' : ''}
          </p>
        </div>
      </div>

      <div style={{
        padding: '0 16px 16px',
        display: 'flex',
        flexDirection: 'column',
        gap: '6px',
      }}>
        {game.prices.map((price, i) => (
          <PriceRow
            key={i}
            price={price}
            usdRate={usdRate}
            showARS={showARS}
            isBest={i === 0}
          />
        ))}
      </div>
    </div>
  )
}