import { render, screen } from '@testing-library/react'
import EnforcementTable from './components/EnforcementTable'
test('missing or unknown evidence is never rendered as PASS', () => {
  render(<EnforcementTable hosts={[{ host: 'X', advisory: undefined, checked: 'UNKNOWN', blocking: 'FAIL' }]} />)
  expect(document.querySelectorAll('[data-verdict="PASS"]').length).toBe(0)
  expect(screen.getAllByText(/Unknown/).length).toBe(2)
  expect(screen.getByText('Not enforced')).toBeInTheDocument()
})
test('PASS renders only when supplied', () => {
  render(<EnforcementTable hosts={[{ host: 'X', advisory: 'PASS', checked: 'UNKNOWN', blocking: 'UNKNOWN' }]} />)
  expect(document.querySelectorAll('[data-verdict="PASS"]').length).toBe(1)
})
