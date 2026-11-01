import { getCollection } from "astro:content";
import { OGImageRoute } from "astro-og-canvas";

// One OpenGraph image per docs page. Keys are the content-entry IDs, so the
// generated path for a page is `/open-graph/<id>.png` — the Head override
// references exactly that URL.
const collectionEntries = await getCollection("docs");

const pages = Object.fromEntries(
  collectionEntries.map(({ id, data }) => [
    id,
    { title: data.title, description: data.description },
  ]),
);

export const { getStaticPaths, GET } = await OGImageRoute({
  pages,

  getImageOptions: (_path, page) => ({
    title: page.title,
    description: page.description,
    bgGradient: [
      [17, 15, 26],
      [38, 26, 66],
    ],
    border: {
      color: [139, 92, 246],
      width: 12,
    },
    font: {
      title: {
        color: [244, 242, 255],
        size: 64,
      },
      description: {
        color: [167, 158, 199],
        size: 34,
      },
    },
    padding: 72,
  }),
});
