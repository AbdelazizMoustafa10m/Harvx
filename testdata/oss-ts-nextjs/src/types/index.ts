/** Represents a blog post. */
export interface Post {
  id: string;
  title: string;
  slug: string;
  excerpt: string;
  content?: string;
  publishedAt: string;
  updatedAt?: string;
  readingTime: number;
  tags: string[];
  author?: Author;
}

/** Represents an author of blog posts. */
export interface Author {
  name: string;
  avatar?: string;
  bio?: string;
  links?: SocialLink[];
}

/** A social media or external link. */
export interface SocialLink {
  platform: 'github' | 'twitter' | 'linkedin' | 'website';
  url: string;
}

/** Configuration for pagination. */
export interface PaginationConfig {
  page: number;
  perPage: number;
  total: number;
}

/** API response wrapper for paginated results. */
export interface PaginatedResponse<T> {
  data: T[];
  pagination: PaginationConfig;
  hasMore: boolean;
}

/** Site-wide metadata used in head tags. */
export interface SiteMetadata {
  title: string;
  description: string;
  siteUrl: string;
  ogImage?: string;
}