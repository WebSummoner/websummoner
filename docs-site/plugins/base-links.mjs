/**
 * Prefix root-relative links and assets with the site's `base`.
 *
 * Astro rewrites the links it generates itself (sidebar, pagination), but a
 * root-relative link written by hand in Markdown — `[flags](/reference/cli-flags/)`
 * or `<img src="/img/thing.gif">` — is passed through untouched. On a project
 * GitHub Pages site served from `<org>.github.io/<repo>/` those all 404.
 *
 * Prefixing them here keeps the source readable (authors write `/reference/…`,
 * not `/repo/reference/…`) and keeps the base a single setting: change
 * `base` in astro.config.mjs and every link follows.
 *
 * Dependency-free on purpose — it walks the hast tree directly rather than
 * pulling in unist-util-visit.
 */
const ATTRS = ['href', 'src', 'poster'];

export default function rehypeBaseLinks({ base = '/' } = {}) {
  const prefix = base.endsWith('/') ? base.slice(0, -1) : base;

  // Nothing to do when the site is served from the domain root.
  if (prefix === '') return () => {};

  const rewrite = (value) => {
    if (typeof value !== 'string') return value;
    // Leave protocol-relative (//host), absolute URLs, anchors and mailto
    // alone; only root-relative paths need the base. Skip anything already
    // prefixed so re-runs stay idempotent.
    if (!value.startsWith('/') || value.startsWith('//')) return value;
    if (value === prefix || value.startsWith(`${prefix}/`)) return value;
    return prefix + value;
  };

  const walk = (node) => {
    if (node.type === 'element' && node.properties) {
      for (const attr of ATTRS) {
        if (attr in node.properties) {
          node.properties[attr] = rewrite(node.properties[attr]);
        }
      }
    }
    // JSX written directly in .mdx (<img src="/img/…" />) is not a hast
    // element — it keeps its attributes in an array — so it needs its own pass.
    if (node.type === 'mdxJsxFlowElement' || node.type === 'mdxJsxTextElement') {
      for (const attr of node.attributes ?? []) {
        if (attr.type === 'mdxJsxAttribute' && ATTRS.includes(attr.name)) {
          attr.value = rewrite(attr.value);
        }
      }
    }
    if (Array.isArray(node.children)) node.children.forEach(walk);
  };

  return (tree) => walk(tree);
}
