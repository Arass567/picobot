export default function KpiCard({ label, value, tone = 'default', hint }) {
  const toneClass = tone === 'success' ? 'text-emerald-400' : 'text-slate-100'
  return (
    <article className="group relative overflow-hidden rounded-2xl glass-panel glass-panel-hover p-6 transition-all">
      {/* Background Decor */}
      <div className={`absolute -right-8 -top-8 h-32 w-32 rounded-full blur-3xl transition-all duration-500 group-hover:scale-110 opacity-20 ${tone === 'success' ? 'bg-emerald-500' : 'bg-slate-500'}`} />

      <div className="relative z-10">
        <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-400 group-hover:text-slate-300 transition-colors">{label}</p>
        <p className={`mt-3 text-3xl font-bold tracking-tight ${toneClass}`}>{value}</p>
        {hint ? <p className="mt-2 text-xs text-slate-500">{hint}</p> : null}
      </div>
    </article>
  )
}
