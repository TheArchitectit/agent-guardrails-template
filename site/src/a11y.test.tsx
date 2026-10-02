import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import axe from 'axe-core'
import App from './App'
const routes = ['/', '/story', '/how-it-works', '/install', '/compatibility', '/policies', '/roadmap', '/security', '/community']
test.each(routes)('no serious axe violations on %s', async route => {
  const { container } = render(<MemoryRouter initialEntries={[route]}><App /></MemoryRouter>)
  const res = await axe.run(container, { rules: { 'color-contrast': { enabled: false }, region: { enabled: false } } })
  const bad = res.violations.filter(v => v.impact === 'serious' || v.impact === 'critical')
  expect(bad.map(v => `${v.id}: ${v.help}`)).toEqual([])
})
