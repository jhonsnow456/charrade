import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import Score from './Score';

const mockRoom = {
  id: 'abc123',
  hostId: 'p1',
  phase: 'finished' as const,
  players: [
    { id: 'p1', name: 'Alice', avatar: 'avatar-1', score: 3 },
    { id: 'p2', name: 'Bob', avatar: 'avatar-2', score: 1 },
    { id: 'p3', name: 'Carol', avatar: 'avatar-3', score: 3 },
  ],
  round: null,
};

function renderScore() {
  return render(
    <BrowserRouter>
      <Score />
    </BrowserRouter>,
  );
}

describe('Score', () => {
  it('fetches the room and shows players sorted by score', async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({ room: mockRoom }),
    })) as unknown as typeof fetch;

    renderScore();

    await waitFor(() => expect(screen.getByText('Final Scores')).toBeInTheDocument());

    const rows = screen.getAllByRole('row');
    expect(rows).toHaveLength(4); // header + 3 players
    expect(rows[1]).toHaveTextContent('Alice');
    expect(rows[1]).toHaveTextContent('3');
    expect(rows[2]).toHaveTextContent('Carol');
    expect(rows[2]).toHaveTextContent('3');
    expect(rows[3]).toHaveTextContent('Bob');
    expect(rows[3]).toHaveTextContent('1');
  });

  it('highlights tied winners', async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({ room: mockRoom }),
    })) as unknown as typeof fetch;

    renderScore();

    await waitFor(() => expect(screen.getByText('Winners: Alice, Carol')).toBeInTheDocument());
  });

  it('shows back-to-home button', async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({ room: mockRoom }),
    })) as unknown as typeof fetch;

    renderScore();

    await waitFor(() => {
      const links = screen.getAllByText('Back to home');
      expect(links.length).toBeGreaterThanOrEqual(1);
    });
  });

  it('shows error when fetch fails', async () => {
    globalThis.fetch = vi.fn(async () => ({
      ok: false,
      status: 404,
    })) as unknown as typeof fetch;

    renderScore();

    await waitFor(() => expect(screen.getByText(/failed to load scores/i)).toBeInTheDocument());
  });
});
