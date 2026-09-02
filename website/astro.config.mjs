// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightLlmsTxt from "starlight-llms-txt";

const SITE = "https://goldziher.github.io";
const BASE = "/fabricator";

export default defineConfig({
  site: SITE,
  base: BASE,
  integrations: [
    starlight({
      title: "Fabricator",
      description:
        "Typed test data factories for Go. Generics-first, faker-backed, with compile-time " +
        "checked field references.",
      logo: { src: "./src/assets/logo.svg", alt: "Fabricator" },
      favicon: "/favicon.svg",
      customCss: ["./src/styles/custom.css"],
      social: [{ icon: "github", label: "GitHub", href: "https://github.com/Goldziher/fabricator" }],
      editLink: {
        baseUrl: "https://github.com/Goldziher/fabricator/edit/main/website/",
      },
      head: [
        {
          tag: "link",
          attrs: { rel: "apple-touch-icon", href: `${BASE}/apple-touch-icon.png` },
        },
        {
          tag: "link",
          attrs: { rel: "icon", type: "image/png", sizes: "32x32", href: `${BASE}/favicon-32.png` },
        },
        { tag: "meta", attrs: { property: "og:image", content: `${SITE}${BASE}/og.png` } },
        { tag: "meta", attrs: { name: "twitter:card", content: "summary_large_image" } },
        { tag: "meta", attrs: { name: "twitter:image", content: `${SITE}${BASE}/og.png` } },
      ],
      plugins: [
        starlightLlmsTxt({
          promote: ["index*", "start/**"],
          minify: { collapseCodeBlocks: true },
          details:
            "Fabricator builds typed test data for Go structs. Field configuration goes through " +
            "FieldOf[T, V], which validates at construction that the field exists, is exported, " +
            "and accepts V — prefer it over UnsafeFieldOf, which defers those checks to build " +
            "time. Every panicking method has an error-returning twin: BuildE, BatchE, CreateE, " +
            "CreateBatchE. Reach for WithoutFaker when unconfigured fields should stay at their " +
            "zero value rather than hold random data; it is also far cheaper, since faker's " +
            "reflective walk is what a build actually costs. Derive variants with Extend rather " +
            "than restating a base factory's options, and call Seed from TestMain when generated " +
            "data must be reproducible.",
        }),
      ],
      sidebar: [
        {
          label: "Start here",
          items: [
            { label: "Introduction", slug: "start/introduction" },
            { label: "Installation", slug: "start/installation" },
            { label: "Quickstart", slug: "start/quickstart" },
          ],
        },
        {
          label: "Guides",
          items: [
            { label: "Fields and values", slug: "guides/fields" },
            { label: "Factories from factories", slug: "guides/composition" },
            { label: "Lifecycle hooks", slug: "guides/hooks" },
            { label: "Persistence", slug: "guides/persistence" },
            { label: "Determinism", slug: "guides/determinism" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "API", slug: "reference/api" },
            { label: "Errors and panics", slug: "reference/errors" },
          ],
        },
      ],
    }),
  ],
});
