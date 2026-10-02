import { NavLink, Outlet } from 'react-router-dom'
import { useEffect, useState } from 'react'
const LINKS: [string, string][] = [['/', 'Home'], ['/story', 'Story'], ['/how-it-works', 'How it works'], ['/install', 'Install'], ['/compatibility', 'Compatibility'], ['/policies', 'Policies'], ['/roadmap', 'Roadmap'], ['/security', 'Security'], ['/community', 'Community']]
export default function Layout() {
  const [theme, setTheme] = useState<'system' | 'light' | 'dark'>('system')
  useEffect(() => { document.documentElement.dataset.theme = theme }, [theme])
  return (
    <>
      <a className="skip" href="#main">Skip to content</a>
      <header><nav aria-label="Main">
        <strong>Agent Guardrails</strong>
        <ul>{LINKS.map(([to, label]) => <li key={to}><NavLink to={to} end={to === '/'}>{label}</NavLink></li>)}</ul>
        <label>Theme <select value={theme} onChange={e => setTheme(e.target.value as typeof theme)}>
          <option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></select></label>
      </nav></header>
      <main id="main"><Outlet /></main>
      <footer>
        <p>License: BSD-3-Clause. Honest-claims policy: every protection claim links to a test or evidence record, and missing evidence reads "Not enforced". Report security issues privately via the repository security policy.</p>
      </footer>
    </>
  )
}
