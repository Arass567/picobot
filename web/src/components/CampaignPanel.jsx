import { useState } from 'react'
import { X, Send } from 'lucide-react'

const defaultForm = {
  slot_time: '',
  offer_label: 'Offre express anti-annulation',
  offer_type: 'discount',
  offer_value: '-20%',
  slot_value_cents: 12000,
  max_recipients: 25,
}

export default function CampaignPanel({ open, onClose, onLaunch, isLoading, error }) {
  const [form, setForm] = useState(defaultForm)

  if (!open) return null

  const handleSubmit = async (event) => {
    event.preventDefault()
    await onLaunch({
      ...form,
      slot_value_cents: Number(form.slot_value_cents),
      max_recipients: Number(form.max_recipients),
    })
  }

  return (
    <div className="fixed inset-0 z-40 flex justify-end bg-slate-950/60 backdrop-blur-sm transition-opacity">
      <aside className="relative h-full w-full max-w-md overflow-hidden border-l border-white/5 glass-panel p-6 shadow-2xl">
        <div className="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-orange-500/10 blur-3xl"></div>
        <div className="pointer-events-none absolute -bottom-32 -left-32 h-80 w-80 rounded-full bg-emerald-500/10 blur-3xl"></div>

        <div className="relative z-10 flex items-center justify-between">
          <h2 className="text-xl font-bold tracking-tight text-white">Créer une relance express</h2>
          <button type="button" className="rounded-full bg-white/5 p-2 text-slate-400 transition-colors hover:bg-white/10 hover:text-white" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="relative z-10 mt-8 space-y-5">
          <label className="block">
            <span className="mb-1.5 block text-[13px] font-medium text-slate-300">Date et heure</span>
            <input
              required
              type="datetime-local"
              value={form.slot_time}
              onChange={(e) => setForm((prev) => ({ ...prev, slot_time: e.target.value }))}
              className="glass-input w-full rounded-xl px-4 py-2.5 text-slate-100"
            />
          </label>

          <label className="block">
            <span className="mb-1.5 block text-[13px] font-medium text-slate-300">Offre</span>
            <input
              required
              value={form.offer_label}
              onChange={(e) => setForm((prev) => ({ ...prev, offer_label: e.target.value }))}
              className="glass-input w-full rounded-xl px-4 py-2.5 text-slate-100"
            />
          </label>

          <div className="grid grid-cols-2 gap-4">
            <label className="block">
              <span className="mb-1.5 block text-[13px] font-medium text-slate-300">Type</span>
              <select
                value={form.offer_type}
                onChange={(e) => setForm((prev) => ({ ...prev, offer_type: e.target.value }))}
                className="glass-input w-full rounded-xl px-4 py-2.5 text-slate-100 [&>option]:bg-slate-900"
              >
                <option value="discount">Réduction</option>
                <option value="gift">Bonus</option>
              </select>
            </label>

            <label className="block">
              <span className="mb-1.5 block text-[13px] font-medium text-slate-300">Valeur</span>
              <input
                value={form.offer_value}
                onChange={(e) => setForm((prev) => ({ ...prev, offer_value: e.target.value }))}
                className="glass-input w-full rounded-xl px-4 py-2.5 text-slate-100"
              />
            </label>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <label className="block">
              <span className="mb-1.5 block text-[13px] font-medium text-slate-300">CA du créneau (€)</span>
              <input
                required
                min={1}
                type="number"
                value={Math.round(form.slot_value_cents / 100)}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, slot_value_cents: Number(e.target.value || 0) * 100 }))
                }
                className="glass-input w-full rounded-xl px-4 py-2.5 text-slate-100"
              />
            </label>

            <label className="block">
              <span className="mb-1.5 block text-[13px] font-medium text-slate-300">Clients ciblés max</span>
              <input
                min={1}
                type="number"
                value={form.max_recipients}
                onChange={(e) => setForm((prev) => ({ ...prev, max_recipients: Number(e.target.value || 1) }))}
                className="glass-input w-full rounded-xl px-4 py-2.5 text-slate-100"
              />
            </label>
          </div>

          {error ? <p className="rounded-xl border border-red-500/40 bg-red-500/10 p-3 text-sm text-red-200">{error}</p> : null}

          <button
            disabled={isLoading}
            type="submit"
            className="group relative mt-4 w-full overflow-hidden rounded-xl bg-gradient-to-r from-orange-500 to-orange-600 px-4 py-3.5 text-[15px] font-bold text-white shadow-lg shadow-orange-500/25 transition-all hover:scale-[1.01] hover:shadow-orange-500/40 disabled:cursor-not-allowed disabled:opacity-70 disabled:hover:scale-100"
          >
            <div className="absolute inset-0 bg-white/20 opacity-0 transition-opacity group-hover:opacity-100"></div>
            <span className="relative z-10 flex items-center justify-center gap-2">
              {isLoading ? 'Envoi en cours...' : (
                <>Lancer la relance <Send size={16} className="text-orange-200 transition-transform group-hover:translate-x-1 group-hover:-translate-y-1" /></>
              )}
            </span>
          </button>
        </form>
      </aside>
    </div>
  )
}
