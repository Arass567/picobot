const BASE_URL = (import.meta.env.BASE_URL || '/').replace(/\/$/, '')
const DEFAULT_API_BASE = `${BASE_URL || ''}/api`
const API_BASE = (import.meta.env.VITE_API_BASE_URL || DEFAULT_API_BASE).replace(/\/$/, '')
const API_KEY = import.meta.env.VITE_API_KEY || ''

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(API_KEY ? { 'X-API-Key': API_KEY } : {}),
      ...(options.headers || {}),
    },
    ...options,
  })

  const contentType = response.headers.get('content-type') || ''
  const payload = contentType.includes('application/json') ? await response.json() : null

  if (!response.ok) {
    throw new Error(payload?.error || `HTTP ${response.status}`)
  }

  return payload
}

export function getDailyKpis({ from, to }) {
  return request(`/api/v1/kpis/daily?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`)
}

export function createFillSlotCampaign(input) {
  return request('/api/v1/campaigns/fill-slot', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function sendCampaign(campaignId, maxRecipients) {
  return request(`/api/v1/campaigns/${campaignId}/send`, {
    method: 'POST',
    body: JSON.stringify({ max_recipients: maxRecipients }),
  })
}
