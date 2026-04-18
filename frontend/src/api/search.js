import axios from 'axios'

const BASE_URL = 'http://localhost:8080'

export async function searchGame(query) {
  const { data } = await axios.get(`${BASE_URL}/api/search`, {
    params: { q: query }
  })
  return data
}

export function toARS(priceUSD, usdRate) {
  if (!priceUSD || !usdRate) return null
  return priceUSD * usdRate
}

export function formatARS(amount) {
  if (!amount) return '—'
  return new Intl.NumberFormat('es-AR', {
    style: 'currency',
    currency: 'ARS',
    maximumFractionDigits: 0
  }).format(amount)
}

export function formatUSD(amount) {
  if (!amount) return '—'
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 2
  }).format(amount)
}