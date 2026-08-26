import { expect, test } from '@playwright/test'

const results = {
  query: 'Hades',
  usd_rate: 1370,
  official_rate: 1200,
  card_rate: 1452,
  tax_rate: 0.21,
  warnings: [],
  results: [{
    id: 'hades', title: 'Hades', image_url: 'https://cdn.cloudflare.steamstatic.com/steam/apps/1145360/header.jpg',
    best_deal: { store_name: 'Steam', price: 8799, regular: 17599, currency: 'ARS', discount_percent: 50, url: 'https://store.steampowered.com/app/1145360/', on_sale: true, is_regional: true },
    prices: [
      { store_name: 'Steam', price: 8799, regular: 17599, currency: 'ARS', discount_percent: 50, url: 'https://store.steampowered.com/app/1145360/', on_sale: true, is_regional: true },
      { store_name: 'GOG', price: 9.99, regular: 24.99, currency: 'USD', discount_percent: 60, url: 'https://www.gog.com/game/hades', on_sale: true, is_regional: false },
    ],
  }],
}

const globalResults = {
  ...results,
  results: [{
    ...results.results[0],
    best_deal: { store_name: 'Steam', price: 24.99, regular: 24.99, currency: 'USD', discount_percent: 0, url: 'https://store.steampowered.com/app/1145360/', on_sale: false, is_regional: false },
    prices: [
      { store_name: 'Steam', price: 24.99, regular: 24.99, currency: 'USD', discount_percent: 0, url: 'https://store.steampowered.com/app/1145360/', on_sale: false, is_regional: false },
      results.results[0].prices[1],
    ],
  }],
}

const deals = {
  featured: [{ id: 'deal-1', title: 'Balatro', image_url: '', store_name: 'Epic Games Store', price: 6.49, regular: 14.99, currency: 'USD', discount_percent: 57, url: 'https://store.epicgames.com/', score: 84, reasons: ['96% positivas', 'Mínimo histórico'] }],
  free: [{ id: 'deal-2', title: 'Free Favorite', image_url: '', store_name: 'Steam', price: 0, regular: 20, currency: 'USD', discount_percent: 100, url: 'https://store.steampowered.com/', score: 72, reasons: ['Gratis ahora'] }],
  discounts: [{ id: 'deal-3', title: 'Deep Discount', image_url: '', store_name: 'Microsoft Store', price: 2, regular: 20, currency: 'USD', discount_percent: 90, url: 'https://www.microsoft.com/', score: 50, reasons: [] }],
}
const discover = {
  popular: [{ id: '730', title: 'Counter-Strike 2', image_url: '', steam_url: 'https://store.steampowered.com/app/730/', rank: 1 }],
  most_played: [{ id: '570', title: 'Dota 2', image_url: '', steam_url: 'https://store.steampowered.com/app/570/', rank: 2, players: 800000 }],
}

async function mockAPI(page) {
  await page.route('**/api/deals**', (route) => route.fulfill({ json: deals }))
  await page.route('**/api/discover**', (route) => route.fulfill({ json: discover }))
  await page.route('**/api/search**', (route) => route.fulfill({ json: route.request().url().includes('steam_mode=global') ? globalResults : results }))
}

test('search is shareable and shows only the real regional Steam price', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/?q=Hades&steam=regional')
  await expect(page.getByRole('heading', { name: 'Hades', exact: true })).toBeVisible()
  await expect(page.getByText('$\u00a08.799').first()).toBeVisible()
  await expect(page.getByText('regional · ARS')).toBeVisible()
  await expect(page.getByText('USD: final estimado con dólar tarjeta · IVA 21% · ARS: precio publicado por tienda')).toBeVisible()
  await expect(page.getByText('GOG', { exact: true })).toBeVisible()
  await expect(page.getByText('$\u00a014.505').first()).toBeVisible()
})

test('currency and Steam location controls update the rendered price', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/?q=Hades&steam=regional')
  await page.getByRole('button', { name: 'USD', exact: true }).click()
  await expect(page.getByText('$9.99').first()).toBeVisible()
  await page.getByRole('button', { name: 'Global', exact: true }).click()
  await expect(page.getByText('$24.99').first()).toBeVisible()
})

test('Steam rankings open a price comparison', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Populares en Steam' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Más jugados en Steam' })).toBeVisible()
  await page.getByRole('button', { name: /comparar precios de counter-strike 2/i }).click()
  await expect(page.getByRole('heading', { name: 'Counter-Strike 2', exact: true })).toBeVisible()
})

test('offers switch between curated, free and biggest discount views', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await expect(page.getByRole('button', { name: 'Balatro' })).toBeVisible()
  await page.getByRole('tab', { name: /gratis ahora/i }).click()
  await expect(page.getByRole('button', { name: 'Free Favorite' })).toBeVisible()
  await page.getByRole('tab', { name: /más descuento/i }).click()
  await expect(page.getByRole('button', { name: 'Deep Discount' })).toBeVisible()
})

test('mobile layout has no horizontal overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await mockAPI(page)
  await page.goto('/?q=Hades')
  await expect(page.getByRole('heading', { name: 'Hades', exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
})
