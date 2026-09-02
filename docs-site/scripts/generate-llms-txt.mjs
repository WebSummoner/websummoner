// Generates dist/llms.txt after `astro build` — a plain-text index of every
// help page for AI assistants and crawlers, mirroring bigbluebutton.com/docs.
//
// Pages are discovered from the built HTML (so it always matches what the
// site actually serves) and enriched with title/description from the markdown
// frontmatter. Section order follows the sidebar: getting-started,
// in-meeting, troubleshooting, reference, then the landing page.
//
// Called by `npm run build`; never run manually unless debugging.

import { readdirSync, readFileSync, writeFileSync, existsSync } from 'node:fs';
import { join, dirname, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const dist = join(root, 'dist');
const contentDocs = join(root, 'src', 'content', 'docs');

const SITE_URL = 'https://websummoner.riadvice.com/websummoner';
const SECTION_ORDER = ['getting-started', 'guides', 'reference', 'troubleshooting'];
const SECTION_LABELS = {
  '': 'Start here & project pages',
  'getting-started': 'Getting started',
  guides: 'Guides',
  reference: 'Reference',
  troubleshooting: 'Troubleshooting',
};

function walkHtml(dir) {
  let out = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, entry.name);
    if (entry.isDirectory()) out = out.concat(walkHtml(p));
    else if (entry.name === 'index.html') out.push(p);
  }
  return out;
}

function frontmatter(mdPath) {
  if (!existsSync(mdPath)) return {};
  const fm = readFileSync(mdPath, 'utf8').match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!fm) return {};
  const pick = (key) => fm[1].match(new RegExp(`^${key}:\\s*[\"']?(.+?)[\"']?\\s*$`, 'm'))?.[1];
  return { title: pick('title'), description: pick('description') };
}

const htmlFiles = walkHtml(dist);
const pages = htmlFiles.map((file) => {
  const rel = relative(dist, file); // e.g. in-meeting\audio\index.html
  const slug = '/' + rel.split(sep).slice(0, -1).join('/');
  const docDir = slug === '/' ? '' : slug.slice(1);
  const meta =
    frontmatter(join(contentDocs, `${docDir}.md`)).title
      ? frontmatter(join(contentDocs, `${docDir}.md`))
      : frontmatter(join(contentDocs, `${docDir}.mdx`));
  const section = docDir.split('/')[0];
  return {
    slug,
    section: SECTION_ORDER.includes(section) ? section : '',
    title: meta.title || 'Welcome',
    description: meta.description || '',
  };
});

pages.sort((a, b) => {
  if (a.slug === '/') return -1;
  if (b.slug === '/') return 1;
  const sa = SECTION_ORDER.indexOf(a.section);
  const sb = SECTION_ORDER.indexOf(b.section);
  if (sa !== sb) return sa - sb;
  return a.slug.localeCompare(b.slug);
});

const lines = [
  '# WebSummoner docs',
  '',
  '> Guides and references for WebSummoner — a fast Selenium hub that summons',
  '> a fleet of browsers into ephemeral, session-scoped Docker containers. Developed and maintained by RIADVICE.',
  '',
];

let current = Symbol('none');
for (const p of pages) {
  if (p.slug === '/') continue;
  if (p.section !== current) {
    current = p.section;
    lines.push(`## ${SECTION_LABELS[current] ?? current}`);
    lines.push('');
  }
  lines.push(`- [${p.title}](${SITE_URL}${p.slug}/): ${p.description}`);
}
lines.push('');

// The landing page, listed last as the entry point.
const home = pages.find((p) => p.slug === '/');
if (home) {
  lines.push('## Start here', '', `- [${home.title}](${SITE_URL}/): ${home.description}`, '');
}

writeFileSync(join(dist, 'llms.txt'), lines.join('\n'), 'utf8');
console.log(`llms.txt: ${pages.length - 1} pages indexed`);
