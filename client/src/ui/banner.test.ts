import { describe, expect, it } from 'vitest';
import { createHeldBanner } from './banner';

// ADR 0012 test 27: the banner shows after the fade in
describe('createHeldBanner', () => {
  it('shows at once when not held; keeps the last name while held, and shows it on release', () => {
    const shown: string[] = [];
    const banner = createHeldBanner({ show: (name) => shown.push(name) });
    banner.show('World');
    banner.hold();
    banner.show('Sprinkle Meadow');
    banner.show('Doma-1');
    expect(shown).toEqual(['World']);
    banner.release();
    expect(shown).toEqual(['World', 'Doma-1']);
    banner.release();
    expect(shown, 'nothing more to show').toEqual(['World', 'Doma-1']);
  });
});
