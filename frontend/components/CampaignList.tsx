'use client'
import { useState, useEffect, useMemo } from 'react';
import Link from 'next/link';
import { api, Campaign } from '../lib/api';
import { ChevronLeft, ChevronRight, Loader2, AlertCircle, MessageSquare, Smartphone } from 'lucide-react';
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  ColumnDef,
} from '@tanstack/react-table';

const statusColors = {
  draft: 'bg-gray-100 text-gray-700',
  scheduled: 'bg-purple-100 text-purple-700',
  sending: 'bg-yellow-100 text-yellow-700',
  sent: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
};

export function CampaignList() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [channelFilter, setChannelFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');

  const limit = 10;
  const totalPages = Math.ceil(total / limit);

  useEffect(() => {
    loadCampaigns();
  }, [page, statusFilter, channelFilter, searchQuery]);

  const loadCampaigns = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await api.getCampaigns(page, limit, statusFilter, channelFilter);

      // Apply client-side search filtering
      let filteredCampaigns = response.campaigns;
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase();
        filteredCampaigns = response.campaigns.filter(campaign =>
          campaign.name.toLowerCase().includes(query) ||
          campaign.template.toLowerCase().includes(query)
        );
      }

      setCampaigns(filteredCampaigns);
      setTotal(response.total);
    } catch (err) {
      setError('Failed to load campaigns. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  const getDeliveryRate = (campaign: Campaign) => {
    if (campaign.sentMessages === 0) return 0;
    return Math.round((campaign.deliveredMessages / campaign.sentMessages) * 100);
  };

  // Define columns for TanStack Table
  const columns = useMemo<ColumnDef<Campaign>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Campaign Name',
        cell: ({ row }) => (
          <Link
            href={`/campaign/${row.original.id}`}
            className="text-blue-600 hover:text-blue-700"
          >
            {row.original.name}
          </Link>
        ),
      },
      {
        accessorKey: 'channel',
        header: 'Channel',
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            {row.original.channel === 'whatsapp' ? (
              <>
                <MessageSquare className="w-4 h-4 text-green-600" />
                <span className="text-gray-900">WhatsApp</span>
              </>
            ) : (
              <>
                <Smartphone className="w-4 h-4 text-blue-600" />
                <span className="text-gray-900">SMS</span>
              </>
            )}
          </div>
        ),
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) => (
          <span className={`inline-flex px-2 py-1 rounded-full capitalize ${statusColors[row.original.status]}`}>
            {row.original.status}
          </span>
        ),
      },
      {
        accessorKey: 'createdAt',
        header: 'Created',
        cell: ({ row }) => (
          <span className="text-gray-600">{formatDate(row.original.createdAt)}</span>
        ),
      },
      {
        accessorKey: 'messages',
        header: 'Messages',
        cell: ({ row }) => (
          <span className="text-gray-900">
            {row.original.sentMessages.toLocaleString()} / {row.original.totalMessages.toLocaleString()}
          </span>
        ),
      },
      {
        accessorKey: 'deliveryRate',
        header: 'Delivery Rate',
        cell: ({ row }) => {
          const rate = getDeliveryRate(row.original);
          return (
            <div className="flex items-center gap-2">
              <div className="flex-1 bg-gray-200 rounded-full h-2 max-w-[100px]">
                <div
                  className="bg-green-600 h-2 rounded-full transition-all"
                  style={{ width: `${rate}%` }}
                />
              </div>
              <span className="text-gray-900 min-w-[45px]">
                {rate}%
              </span>
            </div>
          );
        },
      },
    ],
    []
  );

  // Initialize TanStack Table
  const table = useReactTable({
    data: campaigns,
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    pageCount: totalPages,
  });

  if (loading && campaigns.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 text-blue-600 animate-spin" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
          <p className="text-gray-900 mb-2">{error}</p>
          <button
            onClick={loadCampaigns}
            className="text-blue-600 hover:text-blue-700"
          >
            Try again
          </button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-gray-900 mb-1">Campaigns</h1>
          <p className="text-gray-600">Manage and monitor your messaging campaigns</p>
        </div>
        <Link
          href="/create"
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          Create Campaign
        </Link>
      </div>

      {/* Search and Filters */}
      <div className="mb-6 flex items-center justify-between gap-4">
        {/* Search Bar */}
        <div className="flex-1 max-w-md relative">
          <input
            type="text"
            placeholder="Search campaigns..."
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              setPage(1);
            }}
            className="w-full px-4 py-2 pl-10 bg-white border border-gray-200 rounded-lg text-gray-900 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors"
          />
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
        </div>

        {/* Filters */}
        <div className="flex gap-3">
          {/* Channel Filter */}
          <div className="relative">
            <select
              value={channelFilter}
              onChange={(e) => {
                setChannelFilter(e.target.value);
                setPage(1);
              }}
              className="appearance-none px-4 py-2 pr-10 bg-white border border-gray-200 rounded-lg text-gray-700 hover:border-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors cursor-pointer"
            >
              <option value="all">All Channels</option>
              <option value="whatsapp">WhatsApp</option>
              <option value="sms">SMS</option>
            </select>
            <div className="absolute inset-y-0 right-0 flex items-center pr-3 pointer-events-none">
              <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </div>
          </div>

          {/* Status Filter */}
          <div className="relative">
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(1);
              }}
              className="appearance-none px-4 py-2 pr-10 bg-white border border-gray-200 rounded-lg text-gray-700 hover:border-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors cursor-pointer capitalize"
            >
              <option value="all">All Status</option>
              <option value="draft">Draft</option>
              <option value="scheduled">Scheduled</option>
              <option value="sending">Sending</option>
              <option value="sent">Sent</option>
              <option value="failed">Failed</option>
            </select>
            <div className="absolute inset-y-0 right-0 flex items-center pr-3 pointer-events-none">
              <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      {/* TanStack Table */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className="px-6 py-3 text-left text-gray-700"
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {table.getRowModel().rows.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="px-6 py-12 text-center text-gray-500">
                  No campaigns found. Create your first campaign to get started.
                </td>
              </tr>
            ) : (
              table.getRowModel().rows.map((row) => (
                <tr key={row.id} className="hover:bg-gray-50 transition-colors">
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="px-6 py-4">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {
        totalPages > 1 && (
          <div className="mt-6 flex items-center justify-between">
            <p className="text-gray-600">
              Showing {(page - 1) * limit + 1} to {Math.min(page * limit, total)} of {total} campaigns
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-2 border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                <ChevronLeft className="w-5 h-5 text-gray-700" />
              </button>
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((pageNum) => (
                <button
                  key={pageNum}
                  onClick={() => setPage(pageNum)}
                  className={`px-4 py-2 rounded-lg transition-colors ${page === pageNum
                    ? 'bg-blue-600 text-white'
                    : 'border border-gray-200 text-gray-700 hover:bg-gray-50'
                    }`}
                >
                  {pageNum}
                </button>
              ))}
              <button
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className="px-3 py-2 border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                <ChevronRight className="w-5 h-5 text-gray-700" />
              </button>
            </div>
          </div>
        )}
    </div>
  );
}
