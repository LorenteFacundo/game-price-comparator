import { formatMoney } from '../api/client'

export default function DealCard({ deal, onSearch }) {
  return (
    <article className="deal-card">
      <a href={deal.url} target="_blank" rel="noopener noreferrer" className="deal-image-link" aria-label={`Ver ${deal.title} en ${deal.store_name}`}>
        {deal.image_url ? <img src={deal.image_url} alt="" width="300" height="140" loading="lazy" /> : <div className="image-fallback" aria-hidden="true">✦</div>}
      </a>
      <div className="deal-content">
        <div className="deal-meta">
          <span>{deal.store_name}</span>
          {deal.is_near_low && <span className="low-badge">cerca del mínimo</span>}
        </div>
        <button className="deal-title" type="button" onClick={() => onSearch(deal.title)}>{deal.title}</button>
        <div className="deal-price-line">
          <strong>{formatMoney(deal.price, deal.currency)}</strong>
          {deal.regular > deal.price && <del>{formatMoney(deal.regular, deal.currency)}</del>}
          <span className="discount">−{deal.discount_percent}%</span>
        </div>
      </div>
    </article>
  )
}
