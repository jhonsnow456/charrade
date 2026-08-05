import { describe, expect, it } from 'vitest';
import { AVATAR_IDS, avatarSrc, isAvatar } from './avatars';

describe('avatars', () => {
  it('exposes 14 avatars', () => {
    expect(AVATAR_IDS).toHaveLength(14);
  });

  it('uses the avatar-N naming convention', () => {
    expect(AVATAR_IDS).toContain('avatar-1');
    expect(AVATAR_IDS).toContain('avatar-14');
    expect(AVATAR_IDS[0]).toBe('avatar-1');
    expect(AVATAR_IDS[13]).toBe('avatar-14');
  });

  it('validates known and unknown ids', () => {
    expect(isAvatar('avatar-7')).toBe(true);
    expect(isAvatar('avatar-15')).toBe(false);
    expect(isAvatar('dog')).toBe(false);
  });

  it('builds a static asset path', () => {
    expect(avatarSrc('avatar-3')).toBe('/avatars/avatar-3.svg');
  });
});
