import axios from 'axios'

export const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15000,
})

function messageFrom(error, fallback) {
  return error.response?.data?.error || error.message || fallback
}

export async function searchGames(query, signal) {
  try {
    const { data } = await api.get('/api/search', { params: { q: query }, signal })
    return data
  } catch (error) {
    if (axios.isCancel(error)) throw error
    throw new Error(messageFrom(error, 'No pudimos buscar ese juego.'), { cause: error })
  }
}

export async function getHealth(signal) {
  try {
    const { data } = await api.get('/api/health', { signal, timeout: 10000 })
    return data
  } catch (error) {
    if (axios.isCancel(error)) throw error
    throw new Error(messageFrom(error, 'La API no está respondiendo.'), { cause: error })
  }
}

export async function getDeals(signal) {
  try {
    const { data } = await api.get('/api/deals', { params: { limit: 12 }, signal, timeout: 19500 })
    return {
      featured: data.featured || [],
      free: data.free || [],
      discounts: data.discounts || [],
      updatedAt: data.updated_at || '',
      warnings: data.warnings || [],
    }
  } catch (error) {
    if (axios.isCancel(error)) throw error
    throw new Error(messageFrom(error, 'No pudimos cargar las ofertas.'), { cause: error })
  }
}

export async function getDiscover(signal) {
  try {
    const { data } = await api.get('/api/discover', { signal })
    return { popular: data.popular || [], mostPlayed: data.most_played || [], updatedAt: data.updated_at || '', warnings: data.warnings || [] }
  } catch (error) {
    if (axios.isCancel(error)) throw error
    throw new Error(messageFrom(error, 'No pudimos cargar los rankings de Steam.'), { cause: error })
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

export function displayFinalMoney(price, preferredCurrency, officialRate, cardRate, taxRate) {
  if (!price?.price || !price?.currency) return { label: 'Sin precio', converted: false, includesTax: false, note: '' }
  const currency = price?.currency?.toUpperCase()
  const totalRate = cardRate > 0 ? cardRate : officialRate * (1 + taxRate)

  if (preferredCurrency === 'USD') {
    return { label: formatMoney(price.price, 'USD'), converted: false, includesTax: false, note: 'precio publicado en USD' }
  }

  if (currency === 'ARS') return { label: formatMoney(price.price, 'ARS'), converted: false, includesTax: false, note: 'precio regional publicado · no se reconvierte' }
  if (currency === 'USD' && totalRate > 0) return { label: formatMoney(price.price * totalRate, 'ARS'), converted: true, includesTax: true, note: `total con IVA ${Math.round(taxRate * 100)}%` }
  return { label: formatMoney(price.price, currency), converted: false, includesTax: false, note: 'cotización no disponible' }
}
