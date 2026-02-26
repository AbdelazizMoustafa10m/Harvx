import React from 'react';

interface FooterLink {
  label: string;
  href: string;
  external?: boolean;
}

const footerLinks: FooterLink[] = [
  { label: 'GitHub', href: 'https://github.com/example/devblog', external: true },
  { label: 'RSS Feed', href: '/feed.xml' },
  { label: 'Privacy', href: '/privacy' },
];

export const Footer: React.FC = () => {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="bg-gray-50 border-t border-gray-200 mt-auto">
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="flex flex-col md:flex-row items-center justify-between">
          <p className="text-gray-500 text-sm">
            &copy; {currentYear} DevBlog. All rights reserved.
          </p>

          <div className="flex space-x-4 mt-4 md:mt-0">
            {footerLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                className="text-gray-500 hover:text-gray-700 text-sm"
                {...(link.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
              >
                {link.label}
              </a>
            ))}
          </div>
        </div>
      </div>
    </footer>
  );
};