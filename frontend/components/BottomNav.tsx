"use client";

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Home, Heart, MessageCircle, User } from 'lucide-react';

export default function BottomNav() {
  const pathname = usePathname();

  const isActive = (path: string) => pathname === path;

  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 pb-safe z-50">
      <div className="flex justify-around items-center h-16">
        <Link href="/" className={`flex flex-col items-center p-2 ${isActive('/') ? 'text-blue-600' : 'text-gray-500'}`}>
          <Home size={24} />
          <span className="text-xs mt-1">Home</span>
        </Link>
        <Link href="/couple" className={`flex flex-col items-center p-2 ${isActive('/couple') ? 'text-blue-600' : 'text-gray-500'}`}>
          <Heart size={24} />
          <span className="text-xs mt-1">Couple</span>
        </Link>
        <Link href="/chat" className={`flex flex-col items-center p-2 ${isActive('/chat') ? 'text-blue-600' : 'text-gray-500'}`}>
          <MessageCircle size={24} />
          <span className="text-xs mt-1">Chat</span>
        </Link>
        <Link href="/profile" className={`flex flex-col items-center p-2 ${isActive('/profile') ? 'text-blue-600' : 'text-gray-500'}`}>
          <User size={24} />
          <span className="text-xs mt-1">Profile</span>
        </Link>
      </div>
    </nav>
  );
}
