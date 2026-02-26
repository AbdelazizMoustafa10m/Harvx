import type { GetStaticProps, NextPage } from 'next';
import Head from 'next/head';
import { Layout } from '@/components/Layout';
import type { Post } from '@/types';
import { formatDate } from '@/lib/utils';

interface HomeProps {
  posts: Post[];
}

const Home: NextPage<HomeProps> = ({ posts }) => {
  return (
    <Layout>
      <Head>
        <title>DevBlog - A Developer Blog</title>
        <meta name="description" content="Articles about web development and programming" />
      </Head>

      <main className="max-w-4xl mx-auto px-4 py-8">
        <h1 className="text-4xl font-bold mb-8">Latest Posts</h1>

        <div className="space-y-6">
          {posts.map((post) => (
            <article key={post.id} className="border-b pb-6">
              <h2 className="text-2xl font-semibold hover:text-blue-600">
                <a href={`/posts/${post.slug}`}>{post.title}</a>
              </h2>
              <p className="text-gray-500 text-sm mt-1">
                {formatDate(post.publishedAt)} &middot; {post.readingTime} min read
              </p>
              <p className="text-gray-700 mt-2">{post.excerpt}</p>
              <div className="flex gap-2 mt-3">
                {post.tags.map((tag) => (
                  <span
                    key={tag}
                    className="bg-gray-100 text-gray-600 px-2 py-1 rounded text-xs"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </article>
          ))}
        </div>
      </main>
    </Layout>
  );
};

export const getStaticProps: GetStaticProps<HomeProps> = async () => {
  const posts: Post[] = [
    {
      id: '1',
      title: 'Getting Started with Next.js',
      slug: 'getting-started-nextjs',
      excerpt: 'Learn the basics of Next.js and build your first application.',
      publishedAt: '2024-01-15T00:00:00Z',
      readingTime: 5,
      tags: ['nextjs', 'react', 'typescript'],
    },
    {
      id: '2',
      title: 'TypeScript Best Practices',
      slug: 'typescript-best-practices',
      excerpt: 'Improve your TypeScript code with these proven patterns.',
      publishedAt: '2024-01-10T00:00:00Z',
      readingTime: 8,
      tags: ['typescript', 'javascript'],
    },
  ];

  return {
    props: { posts },
    revalidate: 3600,
  };
};

export default Home;