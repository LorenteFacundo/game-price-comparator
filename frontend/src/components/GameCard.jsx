import PriceRow from './PriceRow'
import { formatARS, formatUSD, getDisplayPrice } from '../api/search'

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

  const bestAmount = getDisplayPrice(game.best_deal, showARS, usdRate)
  const bestPrice = showARS ? formatARS(bestAmount) : formatUSD(bestAmount)

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
            Desde {bestPrice} - {game.prices.length} tienda{game.prices.length !== 1 ? 's' : ''}
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
            isBest={
              game.best_deal?.store_name === price.store_name &&
              game.best_deal?.url === price.url
            }
          />
        ))}
      </div>
    </div>
  )
}
