export const AVATAR_IDS = Array.from({ length: 14 }, (_, i) => `avatar-${i + 1}`);

export function isAvatar(id: string): boolean {
  return AVATAR_IDS.includes(id);
}

export function avatarSrc(id: string): string {
  return `/avatars/${id}.svg`;
}
