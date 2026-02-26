import type { NextPage } from 'next';
import Head from 'next/head';
import { Layout } from '@/components/Layout';

const About: NextPage = () => {
  return (
    <Layout>
      <Head>
        <title>About - DevBlog</title>
      </Head>

      <main className="max-w-4xl mx-auto px-4 py-8">
        <h1 className="text-4xl font-bold mb-6">About</h1>

        <div className="prose prose-lg">
          <p>
            DevBlog is a space for sharing knowledge about web development,
            programming best practices, and modern tooling.
          </p>

          <h2>Tech Stack</h2>
          <ul>
            <li>Next.js for server-side rendering</li>
            <li>TypeScript for type safety</li>
            <li>Tailwind CSS for styling</li>
          </ul>

          <h2>Contact</h2>
          <p>
            Have questions or suggestions? Reach out via{' '}
            <a href="https://github.com/example/devblog" className="text-blue-600 hover:underline">
              GitHub
            </a>.
          </p>
        </div>
      </main>
    </Layout>
  );
};

export default About;