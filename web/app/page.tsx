'use client';

import React from 'react';
import {useSearchParams} from 'next/navigation';
import App from '@/App';
import {GameProvider} from '@/store/GameProvider';
import {MockGameProvider} from '@/mocks/MockGameProvider';
import type {MockScenario} from '@/mocks/mockState';

function AppShell() {
  const searchParams = useSearchParams();
  const mockParam = searchParams.get('mock');
  const envUseMocks = process.env.NEXT_PUBLIC_USE_MOCKS === 'true';
  const envScenario = process.env.NEXT_PUBLIC_MOCK_SCENARIO as MockScenario | undefined;
  const useMocks = envUseMocks || mockParam === '1' || Boolean(mockParam);
  const scenario = (mockParam && mockParam !== '1' ? mockParam : envScenario) as MockScenario | undefined;

  if (useMocks) {
    return (
      <MockGameProvider scenario={scenario}>
        <App />
      </MockGameProvider>
    );
  }

  return (
    <GameProvider>
      <App />
    </GameProvider>
  );
}

export default function Page() {
  return (
    <React.Suspense fallback={null}>
      <AppShell />
    </React.Suspense>
  );
}
