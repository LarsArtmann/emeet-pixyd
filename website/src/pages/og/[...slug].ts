import { getCollection } from "astro:content";
import { OGImageRoute } from "astro-og-canvas";
import { siteConfig } from "../../data/config";

const docs = await getCollection("docs");
const docPages = Object.fromEntries(docs.map(({ data, id }) => [id, { data }]));

const pages = {
  ...docPages,
  home: {
    data: {
      title: siteConfig.name,
      description:
        "Auto-activation daemon for the EMEET PIXY webcam on Linux. Face tracking, privacy mode, audio switching.",
    },
  },
};

export const { getStaticPaths, GET } = await OGImageRoute({
  pages,
  param: "slug",
  getImageOptions: (_path, page) => ({
    title: page.data.title,
    description: page.data.description,
    bgGradient: [[12, 10, 9]],
    border: { color: [139, 92, 246], width: 4 },
    padding: 80,
  }),
});
