/**
 * Web-client login helpers (login form is outside formN_ DOM used by fillFields).
 */

import { readSectionsScript } from './dom.mjs';

/** @param {import('playwright').Page} page */
export async function isWebClientLoggedIn(page) {
  const sections = await page.evaluate(readSectionsScript());
  return sections.some((s) => (s.name || '').trim().length > 0);
}

/** @param {import('playwright').Page} page */
export async function submitWebClientLogin(page, user, password = '') {
  const clicked = await page.evaluate(({ user, password }) => {
    const norm = (s) => (s || '').trim().replace(/\u00a0/g, ' ');
    const inputs = [...document.querySelectorAll('input')].filter((i) => i.offsetWidth > 0);
    if (inputs.length < 1) return { ok: false, reason: 'no_inputs' };

    const setValue = (el, value) => {
      el.focus();
      el.value = value;
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
    };

    let userInput = inputs.find((i) => /пользов/i.test(norm(i.placeholder) + norm(i.getAttribute('aria-label'))));
    let passInput = inputs.find((i) => /парол/i.test(norm(i.placeholder) + norm(i.getAttribute('aria-label'))));

    if (!userInput && inputs.length >= 1) userInput = inputs[0];
    if (!passInput && inputs.length >= 2) passInput = inputs[1];

    if (userInput) setValue(userInput, user);
    if (passInput) setValue(passInput, password || '');

    const btn = [...document.querySelectorAll('.btnText, a.press, button')]
      .find((e) => /войти/i.test(norm(e.innerText || e.textContent)));
    if (!btn) return { ok: false, reason: 'no_login_button' };
    btn.click();
    return { ok: true };
  }, { user, password });

  if (!clicked.ok) {
    throw new Error(`submitWebClientLogin: ${clicked.reason}`);
  }
}

/** @param {import('playwright').Page} page */
export async function waitForWebClientMainUI(page, timeoutMs = 90_000) {
  const t0 = Date.now();
  while (Date.now() - t0 < timeoutMs) {
    if (await isWebClientLoggedIn(page)) return true;
    await page.waitForTimeout(1000);
  }
  return false;
}
