import { displayFinalMoney } from '../api/client'

export default function PriceRow({ gameTitle, price, preferredCurrency, officialRate, cardRate, taxRate, isBest, onReport }) {
  const shown = displayFinalMoney(price, preferredCurrency, officialRate, cardRate, taxRate)
  const regular = price.regular > price.price ? displayFinalMoney({ ...price, price: price.regular }, preferredCurrency, officialRate, cardRate, taxRate) : null
  return (
    <li className={`price-row ${isBest ? 'is-best' : ''}`}>
      <div className="store-identity">
        <span className="store-dot" aria-hidden="true" />
        <div>
          <strong>{price.store_name}</strong>
          <span>{price.is_regional ? 'regional · ARS' : price.currency}</span>
        </div>
      </div>
      <div className="row-price">
        {isBest && <span className="best-badge">Mejor precio</span>}
        <strong>{shown.label}</strong>
        {regular && <del>{regular.label}</del>}
        <small>{shown.note}</small>
      </div>
      <div className="price-actions"><a className="store-link" href={price.url} target="_blank" rel="noopener noreferrer">Ir a tienda <span aria-hidden="true">↗</span></a><button className="report-link" type="button" onClick={() => onReport({ gameTitle, store: price.store_name, displayedPrice: shown.label, sourceURL: price.url })}>Reportar precio</button></div>
    </li>
  )
}
