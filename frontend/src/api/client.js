import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
  timeout: 15000,
})

function messageFrom(error, fallback) {
  return error.response?.data?.error || error.message || fallback
}

export async function searchGames(query, steamMode, signal) {
  try {
    const { data } = await api.get('/api/search', { params: { q: query, steam_mode: steamMode }, signal })
    return data
  } catch (error) {
    if (axios.isCancel(error)) throw error
    throw new Error(messageFrom(error, 'No pudimos buscar ese juego.'), { cause: error })
  }
}

export async function getDeals(signal) {
  try {
    const { data } = await api.get('/api/deals', { params: { limit: 12 }, signal })
    return data.deals || []
  } catch (error) {
    if (axios.isCancel(error)) throw error
    throw new Error(messageFrom(error, 'No pudimos cargar las ofertas.'), { cause: error })
  }
}

export function formatMoney(amount, currency) {
  if (!Number.isFinite(amount) || amount <= 0) return 'Sin precio'
  try {
    return new Intl.NumberFormat(currency === 'ARS' ? 'es-AR' : 'en-US', {
      style: 'currency',
      currency: currency || 'USD',
      maximumFractionDigits: currency === 'ARS' ? 0 : 2,
    }).format(amount)
  } catch {
    return `${currency || ''} ${amount.toFixed(2)}`.trim()
  }
}

export function displayMoney(price, preferredCurrency, usdRate) {
  if (!price?.price || !price?.currency) return { label: 'Sin precio', converted: false }
  const currency = price.currency.toUpperCase()
  if (currency === preferredCurrency) return { label: formatMoney(price.price, currency), converted: false }
  if (currency === 'USD' && preferredCurrency === 'ARS' && usdRate > 0) return { label: formatMoney(price.price * usdRate, 'ARS'), converted: true }
  if (currency === 'ARS' && preferredCurrency === 'USD' && usdRate > 0) return { label: formatMoney(price.price / usdRate, 'USD'), converted: true }
  return { label: formatMoney(price.price, currency), converted: false }
}
