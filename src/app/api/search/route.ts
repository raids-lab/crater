import { source } from "@/lib/source";
import { createFromSource } from "fumadocs-core/search/server";
import { createTokenizer } from "@orama/tokenizers/mandarin";

// it should be cached forever
export const revalidate = false;

export const { staticGET: GET } = createFromSource(source, undefined, {
  localeMap: {
    // you can customise search configs for specific locales, like:
    // [locale]: Orama options

    cn: {
      components: {
        tokenizer: createTokenizer(),
      },
      search: {
        threshold: 0,
        tolerance: 0,
      },
    },
    "custom-locale": "english",
  },
});
