import { displayMoney, formatMoney } from '../api/client'

export default function PriceRow({ price, preferredCurrency, usdRate, isBest }) {
  const shown = displayMoney(price, preferredCurrency, usdRate)
  const regular = price.regular > price.price ? formatMoney(price.regular, price.currency) : null
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
        {regular && <del>{regular}</del>}
        {shown.converted && <small>convertido desde {formatMoney(price.price, price.currency)}</small>}
      </div>
      <a className="store-link" href={price.url} target="_blank" rel="noopener noreferrer">Ir a tienda <span aria-hidden="true">↗</span></a>
    </li>
  )
}
