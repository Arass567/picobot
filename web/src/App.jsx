import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, CalendarClock, CalendarPlus, CheckCircle2, Euro, MessageCircleMore, Send, TrendingUp } from 'lucide-react'
import CampaignPanel from './components/CampaignPanel'
import KpiCard from './components/KpiCard'
import { useUIStore } from './store/uiStore'
import { createFillSlotCampaign, getDailyKpis, sendCampaign } from './api/azimutplus'

function asEUR(cents) {
  return new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR' }).format((cents || 0) / 100)
}

function monthBounds() {
  const now = new Date()
  const from = new Date(now.getFullYear(), now.getMonth(), 1)
  const to = new Date(now.getFullYear(), now.getMonth() + 1, 0)
  const fmt = (date) => {
    const y = date.getFullYear()
    const m = String(date.getMonth() + 1).padStart(2, '0')
    const d = String(date.getDate()).padStart(2, '0')
    return `${y}-${m}-${d}`
  }
  return { from: fmt(from), to: fmt(to) }
}

export default function App() {
  const queryClient = useQueryClient()
  const { panelOpen, setPanelOpen } = useUIStore()
  const [lastCampaign, setLastCampaign] = useState(null)
  const [launchError, setLaunchError] = useState('')

  const range = useMemo(() => monthBounds(), [])
  const kpiQuery = useQuery({
    queryKey: ['daily-kpis', range.from, range.to],
    queryFn: () => getDailyKpis(range),
    refetchInterval: 15000,
  })

  const launchMutation = useMutation({
    mutationFn: async (payload) => {
      const created = await createFillSlotCampaign({
        slot_time: new Date(payload.slot_time).toISOString(),
        offer_label: payload.offer_label,
        offer_type: payload.offer_type,
        offer_value: payload.offer_value,
        slot_value_cents: payload.slot_value_cents,
        max_recipients: payload.max_recipients,
      })
      const sent = await sendCampaign(created.campaign_id, payload.max_recipients)
      return { created, sent }
    },
    onSuccess: (data) => {
      setLastCampaign(data)
      setPanelOpen(false)
      setLaunchError('')
      queryClient.invalidateQueries({ queryKey: ['daily-kpis'] })
    },
    onError: (error) => {
      setLaunchError(error.message || 'Erreur inconnue')
    },
  })

  const kpis = kpiQuery.data || {
    recovered_revenue_cents: 0,
    filled_slots_count: 0,
    sent_messages_count: 0,
    reply_yes_count: 0,
  }

  const isOnline = !kpiQuery.isError

  return (
    <div className="az-grid relative min-h-screen text-slate-50 overflow-hidden">
      {/* Decorative Vectors */}
      <div className="pointer-events-none absolute -top-20 -left-20 h-96 w-96 animate-az-blob rounded-full bg-orange-500/20 mix-blend-screen blur-[120px]"></div>
      <div className="delay-2000 pointer-events-none absolute top-20 -right-20 h-96 w-96 animate-az-blob rounded-full bg-emerald-500/15 mix-blend-screen blur-[120px]"></div>
      <div className="delay-4000 pointer-events-none absolute left-1/3 top-1/2 h-[500px] w-[500px] animate-az-blob rounded-full bg-blue-500/10 mix-blend-screen blur-[150px]"></div>

      <div className="relative z-10 mx-auto w-full max-w-7xl p-4 md:p-8">
        <header className="glass-panel rounded-2xl p-6 md:p-8">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-orange-400/90">AzimutPlus · Aperçu opérateur</p>
              <h1 className="mt-2 text-3xl font-bold tracking-tight md:text-4xl lg:text-5xl">Remplir un trou de planning en  <span className="text-transparent bg-clip-text bg-gradient-to-r from-orange-400 to-orange-600">moins d’une minute</span></h1>
              <p className="mt-4 max-w-2xl text-[15px] leading-relaxed text-slate-300">
                Quand un client annule, tu cliques, AzimutPlus relance les bons contacts, et le premier “OUI” récupère ton chiffre d’affaires.
              </p>
            </div>

            <div className="flex items-center gap-4">
              <a
                href="https://azimutcode.com"
                className="inline-flex items-center justify-center rounded-xl border border-slate-700/70 bg-slate-900/60 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:border-slate-500 hover:text-white"
              >
                Retour AzimutCode
              </a>
              <span className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-[11px] font-medium tracking-wide uppercase shadow-sm ${isOnline ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300 shadow-emerald-500/20' : 'border-amber-500/30 bg-amber-500/10 text-amber-300 shadow-amber-500/20'}`}>
                {isOnline ? <CheckCircle2 size={14} /> : <AlertTriangle size={14} />}
                {isOnline ? 'Backend connecté' : 'Mode aperçu'}
              </span>
              <button
                onClick={() => setPanelOpen(true)}
                className="group relative inline-flex items-center justify-center gap-2 overflow-hidden rounded-xl bg-gradient-to-r from-orange-500 to-orange-600 px-6 py-3.5 text-sm font-bold text-white shadow-lg shadow-orange-500/25 transition-all hover:scale-[1.02] hover:shadow-orange-500/40 active:scale-[0.98]"
              >
                <div className="absolute inset-0 bg-white/20 opacity-0 transition-opacity group-hover:opacity-100"></div>
                <CalendarClock size={18} className="animate-az-float" style={{ animationDuration: '3s' }} />
                Remplir un créneau
              </button>
            </div>
          </div>
        </header>

        <section className="mt-4 grid gap-4 md:grid-cols-3">
          <KpiCard label="CA récupéré ce mois" value={asEUR(kpis.recovered_revenue_cents)} tone="success" hint="Ton indicateur principal" />
          <KpiCard label="Messages envoyés" value={String(kpis.sent_messages_count || 0)} hint="Relances déclenchées" />
          <KpiCard label="Réponses OUI" value={String(kpis.reply_yes_count || 0)} hint="Clients prêts à venir" />
        </section>

        <section className="mt-8 grid gap-4 lg:grid-cols-[1.35fr_1fr]">
          <article className="glass-panel glass-panel-hover rounded-2xl p-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-400">Comment ça marche</p>
            <div className="mt-6 grid gap-4 md:grid-cols-3">
              <div className="group rounded-xl border border-slate-700/50 bg-slate-800/30 p-5 transition-colors hover:bg-slate-800/50 relative overflow-hidden">
                <div className="absolute -right-4 -top-4 h-16 w-16 rounded-full bg-orange-500/10 blur-xl group-hover:bg-orange-500/20 transition-all"></div>
                <div className="mb-3 inline-flex rounded-lg bg-orange-500/10 p-2 text-orange-400 shadow-sm"><CalendarPlus size={20} /></div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-500">Étape 1</p>
                <p className="mt-1 text-sm font-medium text-slate-200">Tu déclares le créneau vide</p>
              </div>
              <div className="group rounded-xl border border-slate-700/50 bg-slate-800/30 p-5 transition-colors hover:bg-slate-800/50 relative overflow-hidden">
                <div className="absolute -right-4 -top-4 h-16 w-16 rounded-full bg-blue-500/10 blur-xl group-hover:bg-blue-500/20 transition-all"></div>
                <div className="mb-3 inline-flex rounded-lg bg-blue-500/10 p-2 text-blue-400 shadow-sm"><Send size={20} /></div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-500">Étape 2</p>
                <p className="mt-1 text-sm font-medium text-slate-200">Relance ciblée sur WhatsApp</p>
              </div>
              <div className="group rounded-xl border border-slate-700/50 bg-slate-800/30 p-5 transition-colors hover:bg-slate-800/50 relative overflow-hidden">
                <div className="absolute -right-4 -top-4 h-16 w-16 rounded-full bg-emerald-500/10 blur-xl group-hover:bg-emerald-500/20 transition-all"></div>
                <div className="mb-3 inline-flex rounded-lg bg-emerald-500/10 p-2 text-emerald-400 shadow-sm"><CheckCircle2 size={20} /></div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-500">Étape 3</p>
                <p className="mt-1 text-sm font-medium text-slate-200">Premier “OUI” = confirmé</p>
              </div>
            </div>

            <div className="mt-6 rounded-xl border border-emerald-500/20 bg-emerald-500/10 p-5 backdrop-blur-sm shadow-inner relative overflow-hidden">
              <div className="absolute top-0 right-0 p-4 opacity-10">
                <svg className="w-16 h-16 text-emerald-500" fill="currentColor" viewBox="0 0 24 24"><path d="M12 2L15.09 8.26L22 9.27L17 14.14L18.18 21.02L12 17.77L5.82 21.02L7 14.14L2 9.27L8.91 8.26L12 2Z" /></svg>
              </div>
              <p className="flex items-center gap-2 text-sm font-medium text-emerald-400"><TrendingUp size={16} /> Résultat attendu</p>
              <p className="mt-1.5 text-[15px] text-emerald-100/90 relative z-10">“J’ai récupéré un créneau à 120€ en quelques minutes.”</p>
            </div>
          </article>

          <article className="glass-panel glass-panel-hover rounded-2xl p-6 flex flex-col justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-400">Dernière action</p>
              {!lastCampaign ? (
                <div className="mt-8 flex flex-col items-center text-center opacity-80">
                  <div className="mb-5 inline-flex items-center justify-center rounded-2xl bg-slate-800/40 p-5 shadow-inner">
                    <svg className="h-12 w-12 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                  </div>
                  <p className="text-[15px] text-slate-300">Aucune campagne lancée.</p>
                  <p className="text-sm mt-1 text-slate-500">Clique sur le bouton orange pour simuler.</p>
                </div>
              ) : (
                <div className="mt-5 space-y-3 rounded-xl border border-slate-700/50 bg-slate-800/30 p-5 text-sm text-slate-200">
                  <p className="flex items-center gap-2 font-medium"><MessageCircleMore size={16} className="text-orange-400" /> Campagne #{lastCampaign.created.campaign_id}</p>
                  <p className="flex justify-between border-t border-slate-700/50 pt-3"><span className="text-slate-400">Messages:</span> <span className="font-semibold">{lastCampaign.sent.sent_count} envoyés</span></p>
                  <p className="flex justify-between"><span className="text-slate-400">Statut:</span> <span className="font-semibold text-emerald-400">{lastCampaign.sent.status}</span></p>
                </div>
              )}
            </div>

            <div className="mt-6 rounded-xl border border-slate-700/50 bg-slate-800/30 p-5 relative overflow-hidden group">
              <div className="absolute -right-4 -top-4 h-24 w-24 rounded-full bg-emerald-500/10 blur-2xl transition-all group-hover:bg-emerald-500/20"></div>
              <p className="text-[11px] font-semibold tracking-wider text-slate-400 uppercase">Lecture business</p>
              <p className="mt-2 flex items-baseline gap-2 text-2xl font-bold tracking-tight text-emerald-400">
                <Euro size={20} className="relative top-0.5" /> {asEUR(kpis.recovered_revenue_cents)}
              </p>
              <p className="mt-1 text-[11px] text-slate-500">Chiffre d’affaires récupéré ce mois</p>
            </div>
          </article>
        </section>

        {kpiQuery.isError ? (
          <p className="mt-4 rounded-xl border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-200">
            Backend non joignable. L’interface reste visible pour la démo, mais les KPI live sont indisponibles.
          </p>
        ) : null}
      </div>

      <CampaignPanel
        open={panelOpen}
        onClose={() => setPanelOpen(false)}
        onLaunch={(payload) => launchMutation.mutateAsync(payload)}
        isLoading={launchMutation.isPending}
        error={launchError}
      />
    </div>
  )
}
