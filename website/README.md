# Fabricator docs site

The [Starlight](https://starlight.astro.build) site published to
<https://goldziher.github.io/fabricator/>.

```shell
pnpm install
pnpm run dev     # local preview
pnpm run build   # static build into dist/
```

Deployed by `.github/workflows/docs.yaml` on every push to `main` that touches
this directory.

Brand assets — the logo, favicons, and social image in `public/` and
`src/assets/` — are generated, not edited by hand. Regenerate them with
`python3 ../scripts/generate_assets.py` from the repository root.
