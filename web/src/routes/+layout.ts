// adapter-static fallback ('index.html') already serves every route from the
// SPA shell; prerender=true would crawl all routes at build time and emit
// identical HTML files that are never used.
export const prerender = false;
export const ssr = false;
