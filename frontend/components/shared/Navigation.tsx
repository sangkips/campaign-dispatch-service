'use client'
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { MessageSquare } from 'lucide-react';

export function Navigation() {
    const pathname = usePathname();

    const isActive = (path: string) => {
        return pathname === path;
    };

    return (
        <nav className="bg-white border-b border-gray-200">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                <div className="flex justify-between h-16">
                    <div className="flex">
                        <Link href="/" className="flex items-center gap-2 px-3 -ml-3">
                            <MessageSquare className="w-6 h-6 text-blue-600" />
                            <span className="text-xl text-gray-900">Campaign Manager</span>
                        </Link>
                        <div className="ml-6 flex gap-4">
                            <Link
                                href="/"
                                className={`inline-flex items-center px-3 border-b-2 ${isActive('/')
                                    ? 'border-blue-500 text-gray-900'
                                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                                    }`}
                            >
                                Campaigns
                            </Link>
                            <Link
                                href="/create"
                                className={`inline-flex items-center px-3 border-b-2 ${isActive('/create')
                                    ? 'border-blue-500 text-gray-900'
                                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                                    }`}
                            >
                                Create Campaign
                            </Link>
                        </div>
                    </div>
                </div>
            </div>
        </nav>
    );
}
