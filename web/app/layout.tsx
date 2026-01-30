import './globals.css';
import type {Metadata} from 'next';
import {Cinzel, Work_Sans} from 'next/font/google';
import {ThemeSync} from '@/components/ThemeSync';

const displayFont = Cinzel({
  subsets: ['latin'],
  variable: '--font-display',
  weight: ['400', '600', '700'],
});

const bodyFont = Work_Sans({
  subsets: ['latin'],
  variable: '--font-body',
  weight: ['400', '500', '600', '700'],
});

export const metadata: Metadata = {
  title: 'ElMakina',
  description: 'Realtime UI for the ElMakina game.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className={`${bodyFont.variable} ${displayFont.variable} antialiased`}>
        <ThemeSync />
        {children}
      </body>
    </html>
  );
}
