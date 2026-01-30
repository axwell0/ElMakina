import {notFound} from 'next/navigation';
import ShowcaseClient from './ShowcaseClient';

export default function ShowcasePage() {
    // Hard gate: never ship internal UI tooling in production.
    if (process.env.NODE_ENV === 'production') {
        notFound();
    }

    return <ShowcaseClient />;
}
