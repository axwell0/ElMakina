import ReplayClient from './ReplayClient';
import type { ReplayPayload } from '@/lib/replay';
import { fetchReplay } from '@/lib/replay';

export const dynamic = 'force-dynamic';

type PageProps = {
    params: Promise<{ matchId: string }>;
    searchParams: Promise<Record<string, string | string[] | undefined>>;
};

export default async function Page({ params, searchParams }: PageProps) {
    const { matchId } = await params;
    const resolvedSearchParams = await searchParams;
    const viewerIdRaw = resolvedSearchParams?.viewer_id;
    const viewerId = typeof viewerIdRaw === 'string' ? viewerIdRaw : Array.isArray(viewerIdRaw) ? viewerIdRaw[0] : null;

    let initialPayload: ReplayPayload | null = null;
    let initialError: string | null = null;

    if (matchId && viewerId) {
        try {
            initialPayload = await fetchReplay(matchId, viewerId);
        } catch (err) {
            initialError = err instanceof Error ? err.message : 'Failed to load replay.';
        }
    }

    return (
        <ReplayClient
            matchId={matchId}
            viewerId={viewerId}
            initialPayload={initialPayload}
            initialError={initialError}
        />
    );
}

