import { listPosts } from '#lib/server/blog/posts.ts';

// The header shows the Blog link only when a published post exists.
export const load = () => ({ hasBlog: listPosts().length > 0 });
