import { expect, test } from '@playwright/test'

const results = {
  query: 'Hades',
  usd_rate: 1370,
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

const deals = { deals: [{ id: 'deal-1', title: 'Balatro', image_url: '', store_name: 'Epic Games Store', price: 6.49, regular: 14.99, currency: 'USD', discount_percent: 57, url: 'https://store.epicgames.com/', is_near_low: true, popularity_rank: 3, matched_stores: ['Steam', 'Epic Games Store'] }] }

async function mockAPI(page) {
  await page.route('**/api/deals**', (route) => route.fulfill({ json: deals }))
  await page.route('**/api/search**', (route) => route.fulfill({ json: results }))
}

test('search is shareable and shows only the real regional Steam price', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/?q=Hades&steam=regional')
  await expect(page.getByRole('heading', { name: 'Hades', exact: true })).toBeVisible()
  await expect(page.getByText('$\u00a08.799').first()).toBeVisible()
  await expect(page.getByText('regional · ARS')).toBeVisible()
  await expect(page.getByText('GOG', { exact: true })).toBeVisible()
})

test('popular offers identify the Steam rank and matched stores', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Populares en oferta' })).toBeVisible()
  await expect(page.getByText('#3 en Steam')).toBeVisible()
  await expect(page.getByText('Epic Games Store · 2 tiendas')).toBeVisible()
})

test('mobile layout has no horizontal overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await mockAPI(page)
  await page.goto('/?q=Hades')
  await expect(page.getByRole('heading', { name: 'Hades', exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
})
