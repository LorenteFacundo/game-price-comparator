import { displayFinalMoney } from '../api/client'

export default function DealCard({ deal, currency, officialRate, cardRate, taxRate, onSearch, onReport, lead = false }) {
	const price = displayFinalMoney(deal, currency, officialRate, cardRate, taxRate)
	const regular = displayFinalMoney({ price: deal.regular, currency: deal.currency }, currency, officialRate, cardRate, taxRate)
	const priceLabel = price.label
  return (
    <article className={`deal-card${lead ? ' deal-card-lead' : ''}`}>
      <a href={deal.url} target="_blank" rel="noopener noreferrer" className="deal-image-link" aria-label={`Ver ${deal.title} en ${deal.store_name}`}>
        {deal.image_url ? <img src={deal.image_url} alt="" width="300" height="140" loading="lazy" /> : <div className="image-fallback" aria-hidden="true">✦</div>}
      </a>
      <div className="deal-content">
        <div className="deal-meta">
          <span>{deal.store_name}</span>
        </div>
        <button className="deal-title" type="button" onClick={() => onSearch(deal.title)}>{deal.title}</button>
        <div className="deal-price-line">
          <strong>{priceLabel}</strong>
		  {deal.regular > deal.price && <del>{regular.label}</del>}
          <span className="discount">−{deal.discount_percent}%</span>
		</div>
		{price.note && <small className="deal-price-note">{price.note}</small>}
        {deal.reasons?.length > 0 && <div className="deal-signals" aria-label="Por qué se destaca">{deal.reasons.slice(0, 2).map((reason) => <span key={reason}>{reason}</span>)}</div>}
        <button className="report-link" type="button" onClick={() => onReport({ gameTitle: deal.title, store: deal.store_name, displayedPrice: priceLabel, sourceURL: deal.url })}>¿Precio incorrecto?</button>
      </div>
    </article>
  )
}
