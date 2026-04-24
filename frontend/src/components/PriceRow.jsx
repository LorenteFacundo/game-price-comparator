import React from 'react'
import { formatARS, formatUSD } from '../api/search'

export default function PriceRow({ price, usdRate, showARS, isBest }) {
  const arsPrice = price.price_ars > 0
    ? price.price_ars
    : (price.price_usd > 0 && usdRate > 0 ? price.price_usd * usdRate : 0)

  const arsRegular = price.regular_ars > 0
    ? price.regular_ars
    : (price.regular_usd > 0 && usdRate > 0 ? price.regular_usd * usdRate : 0)

  const usdPrice = price.price_usd > 0
    ? price.price_usd
    : (arsPrice > 0 && usdRate > 0 ? arsPrice / usdRate : 0)

  const usdRegular = price.regular_usd > 0
    ? price.regular_usd
    : (arsRegular > 0 && usdRate > 0 ? arsRegular / usdRate : 0)

  const sinPrecio = arsPrice === 0 && usdPrice === 0
  const esMundoSteam = price.store_name === 'MundoSteam'

  const displayPrice = showARS ? formatARS(arsPrice) : formatUSD(usdPrice)
  const displayRegular = showARS ? formatARS(arsRegular) : formatUSD(usdRegular)

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      padding: '10px 14px',
      borderRadius: '8px',
      background: esMundoSteam
        ? 'rgba(255,68,68,0.05)'
        : isBest
          ? 'rgba(61,220,132,0.07)'
          : 'var(--surface2)',
      border: `1px solid ${esMundoSteam
        ? 'rgba(255,68,68,0.25)'
        : isBest
          ? 'rgba(61,220,132,0.3)'
          : 'var(--border)'}`,
      gap: '12px',
      opacity: esMundoSteam ? 0.8 : 1,
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flex: 1, minWidth: 0 }}>
        {isBest && !esMundoSteam && (
          <span style={{
            background: 'var(--green)',
            color: '#0a2e17',
            fontSize: '10px',
            fontWeight: 700,
            padding: '2px 7px',
            borderRadius: '20px',
            whiteSpace: 'nowrap',
          }}>MEJOR</span>
        )}
        {esMundoSteam && (
          <span style={{
            background: 'rgba(255,68,68,0.15)',
            color: '#ff4444',
            fontSize: '10px',
            fontWeight: 700,
            padding: '2px 7px',
            borderRadius: '20px',
            whiteSpace: 'nowrap',
          }}>NO REC.</span>
        )}
        {price.is_regional && (
          <span style={{
            background: 'rgba(79,187,255,0.12)',
            color: '#4fbbff',
            fontSize: '10px',
            fontWeight: 600,
            padding: '2px 7px',
            borderRadius: '20px',
            whiteSpace: 'nowrap',
          }}>REGIONAL</span>
        )}
        <span style={{
          fontSize: '14px',
          color: esMundoSteam ? '#ff8888' : isBest ? 'var(--text)' : 'var(--muted)',
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}>
          {price.store_name}
        </span>
        {price.discount_percent > 0 && (
          <span style={{
            background: 'rgba(255,179,64,0.15)',
            color: 'var(--amber)',
            fontSize: '11px',
            fontWeight: 600,
            padding: '2px 7px',
            borderRadius: '20px',
            whiteSpace: 'nowrap',
          }}>-{price.discount_percent}%</span>
        )}
      </div>

      <div style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
        {esMundoSteam ? (
          <div style={{ fontSize: '12px', color: '#ff8888', maxWidth: '140px', textAlign: 'right', lineHeight: 1.3 }}>
            Cuenta compartida
          </div>
        ) : sinPrecio ? (
          <div style={{ fontSize: '13px', color: 'var(--muted)' }}>
            Ver precio
          </div>
        ) : (
          <>
            <div style={{
              fontSize: '16px',
              fontWeight: 700,
              color: isBest ? 'var(--green)' : 'var(--text)',
            }}>
              {displayPrice}
            </div>
            {price.on_sale && arsRegular > arsPrice && (
              <div style={{ fontSize: '11px', color: 'var(--muted)', textDecoration: 'line-through' }}>
                {displayRegular}
              </div>
            )}
          </>
        )}
      </div>

      <a
        href={price.url}
        target="_blank"
        rel="noopener noreferrer"
        style={{
          background: esMundoSteam ? 'rgba(255,68,68,0.1)' : 'var(--surface)',
          border: `1px solid ${esMundoSteam ? 'rgba(255,68,68,0.3)' : 'var(--border2)'}`,
          borderRadius: '8px',
          padding: '6px 12px',
          fontSize: '12px',
          color: esMundoSteam ? '#ff6666' : sinPrecio ? 'var(--accent2)' : 'var(--muted)',
          whiteSpace: 'nowrap',
          fontWeight: sinPrecio ? 600 : 400,
        }}
      >
        Ver tienda
      </a>
    </div>
  )
}
