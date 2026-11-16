import { listPosts } from '#lib/server/blog/posts.ts';

export const load = () => ({ posts: listPosts() });
