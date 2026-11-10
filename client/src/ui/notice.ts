/** Shows a short message over the game, replacing the previous one. Text only, never HTML. */
export function showNotice(host: HTMLElement, text: string): void {
  const notice = host.querySelector<HTMLElement>('.notice') ?? host.appendChild(document.createElement('div'));
  notice.className = 'notice';
  notice.textContent = text;
}
