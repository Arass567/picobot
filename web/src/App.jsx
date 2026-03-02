import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  ArrowRight,
  CalendarClock,
  CalendarPlus,
  CheckCircle2,
  Euro,
  MessageCircleMore,
  Send,
  TrendingUp,
} from 'lucide-react'
import CampaignPanel from './components/CampaignPanel'
import KpiCard from './components/KpiCard'
import { useUIStore } from './store/uiStore'
import { createFillSlotCampaign, getDailyKpis, sendCampaign } from './api/azimutplus'

const DEMO_URL = 'https://calendly.com/assani-azimutcode/30min'

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

function ProspectView({ onOpenOperator }) {
  return (
    <div className="relative z-10 mx-auto w-full max-w-6xl p-4 md:p-8">
      <div className="glass-panel rounded-3xl p-6 md:p-10">
        <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-orange-400/90">AzimutPlus · version démonstration</p>
        <h1 className="mt-3 max-w-4xl text-3xl font-bold tracking-tight md:text-5xl">
          Remplissez automatiquement vos créneaux annulés sur WhatsApp
        </h1>
        <p className="mt-4 max-w-3xl text-base text-slate-300 md:text-lg">
          Un client annule ? En moins d&apos;une minute, AzimutPlus envoie une relance ciblée à vos meilleurs contacts.
          Le premier qui répond <span className="font-semibold text-emerald-300">OUI</span> prend le créneau.
        </p>

        <div className="mt-8 flex flex-wrap gap-3">
          <a
            href={DEMO_URL}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-orange-500 to-orange-600 px-6 py-3.5 text-sm font-bold text-white shadow-lg shadow-orange-500/25 transition hover:scale-[1.01] hover:shadow-orange-500/40"
          >
            Réserver une démo gratuite
            <ArrowRight size={16} />
          </a>
          <button
            type="button"
            onClick={onOpenOperator}
            className="rounded-xl border border-slate-700/70 bg-slate-900/60 px-5 py-3.5 text-sm font-semibold text-slate-200 transition hover:border-slate-500 hover:text-white"
          >
            Accès opérateur
          </button>
          <a
            href="https://azimutcode.com"
            className="rounded-xl border border-slate-700/70 bg-slate-900/60 px-5 py-3.5 text-sm font-semibold text-slate-200 transition hover:border-slate-500 hover:text-white"
          >
            Retour AzimutCode
          </a>
        </div>
      </div>

      <section className="mt-5 grid gap-4 md:grid-cols-3">
        <article className="glass-panel rounded-2xl p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.15em] text-slate-400">1. Le problème</p>
          <p className="mt-2 text-lg font-semibold text-white">Chaque annulation = perte de chiffre d&apos;affaires</p>
          <p className="mt-2 text-sm text-slate-300">Coiffeur, esthéticienne, coach: un trou dans l&apos;agenda coûte immédiatement de l&apos;argent.</p>
        </article>
        <article className="glass-panel rounded-2xl p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.15em] text-slate-400">2. La solution</p>
          <p className="mt-2 text-lg font-semibold text-white">Un bouton, une relance WhatsApp ciblée</p>
          <p className="mt-2 text-sm text-slate-300">Vous cliquez “Remplir un créneau”, AzimutPlus contacte automatiquement les bons clients.</p>
        </article>
        <article className="glass-panel rounded-2xl p-5">
          <p className="text-xs font-semibold uppercase tracking-[0.15em] text-slate-400">3. Le résultat</p>
          <p className="mt-2 text-lg font-semibold text-emerald-300">Le premier “OUI” verrouille le créneau</p>
          <p className="mt-2 text-sm text-slate-300">Vous récupérez du CA qui aurait été perdu, sans relancer manuellement.</p>
        </article>
      </section>

      <section className="mt-5 grid gap-4 lg:grid-cols-[1.2fr_1fr]">
        <article className="glass-panel rounded-2xl p-6">
          <p className="text-xs font-semibold uppercase tracking-[0.15em] text-slate-400">Exemple concret</p>
          <p className="mt-3 text-2xl font-bold">Annulation à 14h00 sur un créneau à 120€</p>
          <p className="mt-2 text-slate-300">AzimutPlus envoie une relance ciblée à 25 clients opt-in WhatsApp.</p>
          <div className="mt-5 rounded-xl border border-emerald-500/20 bg-emerald-500/10 p-4">
            <p className="text-sm font-medium text-emerald-300">Résultat type: créneau repris en quelques minutes.</p>
            <p className="mt-1 text-sm text-emerald-100/80">Objectif business: transformer des pertes sèches en revenu récupéré.</p>
          </div>
        </article>

        <article className="glass-panel rounded-2xl p-6">
          <p className="text-xs font-semibold uppercase tracking-[0.15em] text-slate-400">Prochaine étape</p>
          <p className="mt-3 text-xl font-bold text-white">Voir la démo personnalisée sur votre activité</p>
          <p className="mt-2 text-sm text-slate-300">En 30 min, on simule vos annulations et le CA récupérable sur votre base client.</p>
          <a
            href={DEMO_URL}
            target="_blank"
            rel="noreferrer"
            className="mt-6 inline-flex w-full items-center justify-center rounded-xl bg-orange-500 px-4 py-3 text-sm font-bold text-white shadow-lg shadow-orange-500/25 transition hover:bg-orange-400"
          >
            Réserver ma démo
          </a>
        </article>
      </section>
    </div>
  )
}

function OperatorView({
  onCloseOperator,
  isOnline,
  kpis,
  lastCampaign,
  setPanelOpen,
  kpiQuery,
}) {
  return (
    <div className="relative z-10 mx-auto w-full max-w-7xl p-4 md:p-8">
      <header className="glass-panel rounded-2xl p-6 md:p-8">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-orange-400/90">AzimutPlus · aperçu opérateur</p>
            <h1 className="mt-2 text-3xl font-bold tracking-tight md:text-4xl lg:text-5xl">
              Remplir un trou de planning en{' '}
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-orange-400 to-orange-600">moins d&apos;une minute</span>
            </h1>
            <p className="mt-4 max-w-2xl text-[15px] leading-relaxed text-slate-300">
              Quand un client annule, tu cliques, AzimutPlus relance les bons contacts, et le premier “OUI” récupère ton chiffre d&apos;affaires.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <button
              onClick={onCloseOperator}
              className="rounded-xl border border-slate-700/70 bg-slate-900/60 px-4 py-3 text-sm font-semibold text-slate-200 transition hover:border-slate-500 hover:text-white"
            >
              Retour version simple
            </button>
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
            <div className="rounded-xl border border-slate-700/50 bg-slate-800/30 p-5">
              <div className="mb-3 inline-flex rounded-lg bg-orange-500/10 p-2 text-orange-400 shadow-sm"><CalendarPlus size={20} /></div>
              <p className="text-[11px] font-bold uppercase tracking-wider text-slate-500">Étape 1</p>
              <p className="mt-1 text-sm font-medium text-slate-200">Tu déclares le créneau vide</p>
            </div>
            <div className="rounded-xl border border-slate-700/50 bg-slate-800/30 p-5">
              <div className="mb-3 inline-flex rounded-lg bg-blue-500/10 p-2 text-blue-400 shadow-sm"><Send size={20} /></div>
              <p className="text-[11px] font-bold uppercase tracking-wider text-slate-500">Étape 2</p>
              <p className="mt-1 text-sm font-medium text-slate-200">Relance ciblée sur WhatsApp</p>
            </div>
            <div className="rounded-xl border border-slate-700/50 bg-slate-800/30 p-5">
              <div className="mb-3 inline-flex rounded-lg bg-emerald-500/10 p-2 text-emerald-400 shadow-sm"><CheckCircle2 size={20} /></div>
              <p className="text-[11px] font-bold uppercase tracking-wider text-slate-500">Étape 3</p>
              <p className="mt-1 text-sm font-medium text-slate-200">Premier “OUI” = confirmé</p>
            </div>
          </div>

          <div className="mt-6 rounded-xl border border-emerald-500/20 bg-emerald-500/10 p-5 backdrop-blur-sm shadow-inner">
            <p className="flex items-center gap-2 text-sm font-medium text-emerald-400"><TrendingUp size={16} /> Résultat attendu</p>
            <p className="mt-1.5 text-[15px] text-emerald-100/90">“J&apos;ai récupéré un créneau à 120€ en quelques minutes.”</p>
          </div>
        </article>

        <article className="glass-panel glass-panel-hover rounded-2xl p-6 flex flex-col justify-between">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-400">Dernière action</p>
            {!lastCampaign ? (
              <div className="mt-8 flex flex-col items-center text-center opacity-80">
                <div className="mb-5 inline-flex items-center justify-center rounded-2xl bg-slate-800/40 p-5 shadow-inner">
                  <MessageCircleMore className="h-10 w-10 text-slate-500" />
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

          <div className="mt-6 rounded-xl border border-slate-700/50 bg-slate-800/30 p-5">
            <p className="text-[11px] font-semibold tracking-wider text-slate-400 uppercase">Lecture business</p>
            <p className="mt-2 flex items-baseline gap-2 text-2xl font-bold tracking-tight text-emerald-400">
              <Euro size={20} className="relative top-0.5" /> {asEUR(kpis.recovered_revenue_cents)}
            </p>
            <p className="mt-1 text-[11px] text-slate-500">Chiffre d&apos;affaires récupéré ce mois</p>
          </div>
        </article>
      </section>

      {kpiQuery.isError ? (
        <p className="mt-4 rounded-xl border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-200">
          Backend non joignable. L&apos;interface reste visible pour la démo, mais les KPI live sont indisponibles.
        </p>
      ) : null}
    </div>
  )
}

export default function App() {
  const queryClient = useQueryClient()
  const { panelOpen, setPanelOpen } = useUIStore()
  const [lastCampaign, setLastCampaign] = useState(null)
  const [launchError, setLaunchError] = useState('')
  const [operatorMode, setOperatorMode] = useState(false)

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setOperatorMode(window.location.hash === '#operateur')
    }
  }, [])

  const openOperator = () => {
    setOperatorMode(true)
    if (typeof window !== 'undefined') {
      window.location.hash = 'operateur'
    }
  }

  const closeOperator = () => {
    setOperatorMode(false)
    if (typeof window !== 'undefined') {
      window.history.replaceState(null, '', window.location.pathname + window.location.search)
    }
  }

  const range = useMemo(() => monthBounds(), [])
  const kpiQuery = useQuery({
    queryKey: ['daily-kpis', range.from, range.to],
    queryFn: () => getDailyKpis(range),
    refetchInterval: operatorMode ? 15000 : false,
    enabled: operatorMode,
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

  const isOnline = operatorMode ? !kpiQuery.isError : true

  return (
    <div className="az-grid relative min-h-screen overflow-hidden text-slate-50">
      <div className="pointer-events-none absolute -top-20 -left-20 h-96 w-96 animate-az-blob rounded-full bg-orange-500/20 mix-blend-screen blur-[120px]"></div>
      <div className="delay-2000 pointer-events-none absolute top-20 -right-20 h-96 w-96 animate-az-blob rounded-full bg-emerald-500/15 mix-blend-screen blur-[120px]"></div>
      <div className="delay-4000 pointer-events-none absolute left-1/3 top-1/2 h-[500px] w-[500px] animate-az-blob rounded-full bg-blue-500/10 mix-blend-screen blur-[150px]"></div>

      {operatorMode ? (
        <OperatorView
          onCloseOperator={closeOperator}
          isOnline={isOnline}
          kpis={kpis}
          lastCampaign={lastCampaign}
          setPanelOpen={setPanelOpen}
          kpiQuery={kpiQuery}
        />
      ) : (
        <ProspectView onOpenOperator={openOperator} />
      )}

      <CampaignPanel
        open={operatorMode && panelOpen}
        onClose={() => setPanelOpen(false)}
        onLaunch={(payload) => launchMutation.mutateAsync(payload)}
        isLoading={launchMutation.isPending}
        error={launchError}
      />
    </div>
  )
}
