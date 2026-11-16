import { error } from '@sveltejs/kit';

import { listPosts, renderPost } from '#lib/server/blog/posts.ts';

import type { EntryGenerator, PageServerLoad } from './$types';

export const entries: EntryGenerator = () => listPosts().map(({ slug }) => ({ slug }));

export const load: PageServerLoad = ({ params }) => {
	const post = renderPost({ slug: params.slug });
	if (!post) error(404, 'Post not found');
	return post;
};
