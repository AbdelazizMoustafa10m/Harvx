import React from 'react';
import { Header } from './Header';
import { Footer } from './Footer';

interface LayoutProps {
  children: React.ReactNode;
  className?: string;
}

export const Layout: React.FC<LayoutProps> = ({ children, className = '' }) => {
  return (
    <div className={`min-h-screen flex flex-col ${className}`}>
      <Header />
      <div className="flex-1">{children}</div>
      <Footer />
    </div>
  );
};