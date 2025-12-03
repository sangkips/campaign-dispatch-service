'use client'
import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { api, Campaign, Customer } from '../lib/api';
import {
  ArrowLeft,
  Loader2,
  AlertCircle,
  Send,
  CheckCircle,
  XCircle,
  Clock,
  MessageSquare,
  Smartphone,
  Calendar,
  Users
} from 'lucide-react';

const statusColors = {
  draft: 'bg-gray-100 text-gray-700',
  scheduled: 'bg-purple-100 text-purple-700',
  sending: 'bg-yellow-100 text-yellow-700',
  sent: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
};

export function CampaignDetail() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;
  const [campaign, setCampaign] = useState<Campaign | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Customer selection state
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [selectedCustomerIds, setSelectedCustomerIds] = useState<number[]>([]);
  const [showCustomerModal, setShowCustomerModal] = useState(false);
  const [loadingCustomers, setLoadingCustomers] = useState(false);
  const [sendingCampaign, setSendingCampaign] = useState(false);
  const [sendSuccess, setSendSuccess] = useState<string | null>(null);

  useEffect(() => {
    loadCampaign();
  }, [id]);

  const loadCampaign = async () => {
    if (!id) return;

    setLoading(true);
    setError(null);
    try {
      const data = await api.getCampaign(id);
      if (!data) {
        setError('Campaign not found');
      } else {
        setCampaign(data);
      }
    } catch (err) {
      setError('Failed to load campaign details');
    } finally {
      setLoading(false);
    }
  };

  const loadCustomers = async () => {
    setLoadingCustomers(true);
    try {
      const { customers: customerList } = await api.getCustomers(1, 1000);
      setCustomers(customerList);
    } catch (err) {
      setError('Failed to load customers');
    } finally {
      setLoadingCustomers(false);
    }
  };

  const handleOpenCustomerModal = () => {
    setShowCustomerModal(true);
    loadCustomers();
  };

  const handleToggleCustomer = (customerId: number) => {
    setSelectedCustomerIds(prev =>
      prev.includes(customerId)
        ? prev.filter(id => id !== customerId)
        : [...prev, customerId]
    );
  };

  const handleSelectAll = () => {
    if (selectedCustomerIds.length === customers.length) {
      setSelectedCustomerIds([]);
    } else {
      setSelectedCustomerIds(customers.map(c => c.id));
    }
  };

  const handleSendCampaign = async () => {
    if (!id || selectedCustomerIds.length === 0) return;

    setSendingCampaign(true);
    setSendSuccess(null);
    setError(null);

    try {
      const response = await api.sendCampaign(id, selectedCustomerIds);
      setSendSuccess(`Campaign sent! ${response.messages_queued} messages queued for delivery.`);
      setShowCustomerModal(false);
      setSelectedCustomerIds([]);
      // Reload campaign to get updated stats
      await loadCampaign();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send campaign');
    } finally {
      setSendingCampaign(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'long',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getDeliveryRate = () => {
    if (!campaign || campaign.sentMessages === 0) return 0;
    return Math.round((campaign.deliveredMessages / campaign.sentMessages) * 100);
  };

  const getFailureRate = () => {
    if (!campaign || campaign.sentMessages === 0) return 0;
    return Math.round((campaign.failedMessages / campaign.sentMessages) * 100);
  };

  const getProgress = () => {
    if (!campaign || campaign.totalMessages === 0) return 0;
    return Math.round((campaign.sentMessages / campaign.totalMessages) * 100);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 text-blue-600 animate-spin" />
      </div>
    );
  }

  if (error || !campaign) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
          <p className="text-gray-900 mb-4">{error || 'Campaign not found'}</p>
          <Link
            href="/"
            className="text-blue-600 hover:text-blue-700"
          >
            Back to campaigns
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <button
          onClick={() => router.push('/')}
          className="flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-4 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          Back to campaigns
        </button>

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-gray-900 mb-2">{campaign.name}</h1>
            <div className="flex items-center gap-3 flex-wrap">
              <span className={`inline-flex px-3 py-1 rounded-full capitalize ${statusColors[campaign.status]}`}>
                {campaign.status}
              </span>
              <div className="flex items-center gap-2 text-gray-600">
                {campaign.channel === 'whatsapp' ? (
                  <>
                    <MessageSquare className="w-4 h-4 text-green-600" />
                    <span>WhatsApp</span>
                  </>
                ) : (
                  <>
                    <Smartphone className="w-4 h-4 text-blue-600" />
                    <span>SMS</span>
                  </>
                )}
              </div>
              <span className="text-gray-600">
                Created {formatDate(campaign.createdAt)}
              </span>
              {campaign.scheduledDate && (
                <div className="flex items-center gap-2 text-gray-600">
                  <Calendar className="w-4 h-4" />
                  <span>Scheduled for {formatDate(campaign.scheduledDate)}</span>
                </div>
              )}
            </div>
          </div>

          {/* Send Campaign Button */}
          {(campaign.status === 'draft' || campaign.status === 'scheduled') && (
            <button
              onClick={handleOpenCustomerModal}
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              <Send className="w-4 h-4" />
              Send Campaign
            </button>
          )}
        </div>
      </div>

      {/* Success Message */}
      {sendSuccess && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg flex items-center gap-3">
          <CheckCircle className="w-5 h-5 text-green-600" />
          <p className="text-green-900">{sendSuccess}</p>
        </div>
      )}

      {/* Progress Bar */}
      <div className="bg-white rounded-lg border border-gray-200 p-6 mb-6">
        <div className="flex items-center justify-between mb-2">
          <span className="text-gray-700">Campaign Progress</span>
          <span className="text-gray-900">{getProgress()}%</span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-3">
          <div
            className="bg-blue-600 h-3 rounded-full transition-all"
            style={{ width: `${getProgress()}%` }}
          />
        </div>
        <div className="mt-2 flex justify-between text-gray-600">
          <span>{campaign.sentMessages.toLocaleString()} sent</span>
          <span>{campaign.totalMessages.toLocaleString()} total</span>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-blue-100 rounded-lg">
              <Send className="w-5 h-5 text-blue-600" />
            </div>
            <span className="text-gray-600">Sent</span>
          </div>
          <p className="text-gray-900">{campaign.sentMessages.toLocaleString()}</p>
          <p className="text-gray-500 mt-1">
            {campaign.totalMessages > 0
              ? `${Math.round((campaign.sentMessages / campaign.totalMessages) * 100)}% of total`
              : 'No messages scheduled'
            }
          </p>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-green-100 rounded-lg">
              <CheckCircle className="w-5 h-5 text-green-600" />
            </div>
            <span className="text-gray-600">Delivered</span>
          </div>
          <p className="text-gray-900">{campaign.deliveredMessages.toLocaleString()}</p>
          <p className="text-gray-500 mt-1">
            {getDeliveryRate()}% delivery rate
          </p>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-red-100 rounded-lg">
              <XCircle className="w-5 h-5 text-red-600" />
            </div>
            <span className="text-gray-600">Failed</span>
          </div>
          <p className="text-gray-900">{campaign.failedMessages.toLocaleString()}</p>
          <p className="text-gray-500 mt-1">
            {getFailureRate()}% failure rate
          </p>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-gray-100 rounded-lg">
              <Clock className="w-5 h-5 text-gray-600" />
            </div>
            <span className="text-gray-600">Pending</span>
          </div>
          <p className="text-gray-900">
            {(campaign.totalMessages - campaign.sentMessages).toLocaleString()}
          </p>
          <p className="text-gray-500 mt-1">
            Not yet sent
          </p>
        </div>
      </div>

      {/* Template */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h2 className="text-gray-900 mb-4">Message Template</h2>
        <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
          <code className="text-gray-900 whitespace-pre-wrap break-words">
            {campaign.template}
          </code>
        </div>
        <div className="mt-4 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <p className="text-blue-900">
            <strong>Template Variables:</strong> This template uses variables like{' '}
            <code className="bg-blue-100 px-2 py-1 rounded">{'{first_name}'}</code> that will be
            replaced with actual customer data when messages are sent.
          </p>
        </div>
      </div>

      {/* Customer Selection Modal */}
      {showCustomerModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
            {/* Modal Header */}
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-2xl font-bold text-gray-900">Select Customers</h2>
                  <p className="text-gray-600 mt-1">
                    Choose which customers should receive this campaign
                  </p>
                </div>
                <button
                  onClick={() => setShowCustomerModal(false)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <XCircle className="w-6 h-6" />
                </button>
              </div>
            </div>

            {/* Modal Body */}
            <div className="flex-1 overflow-y-auto p-6">
              {loadingCustomers ? (
                <div className="flex items-center justify-center py-12">
                  <Loader2 className="w-8 h-8 text-blue-600 animate-spin" />
                </div>
              ) : customers.length === 0 ? (
                <div className="text-center py-12">
                  <Users className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                  <p className="text-gray-600">No customers found</p>
                </div>
              ) : (
                <div>
                  {/* Select All */}
                  <div className="mb-4 flex items-center gap-3 p-3 bg-gray-50 rounded-lg">
                    <input
                      type="checkbox"
                      checked={selectedCustomerIds.length === customers.length}
                      onChange={handleSelectAll}
                      className="w-4 h-4 text-blue-600 rounded"
                    />
                    <span className="text-gray-700 font-medium">
                      Select All ({customers.length} customers)
                    </span>
                    {selectedCustomerIds.length > 0 && (
                      <span className="ml-auto text-blue-600 font-medium">
                        {selectedCustomerIds.length} selected
                      </span>
                    )}
                  </div>

                  {/* Customer List */}
                  <div className="space-y-2">
                    {customers.map((customer) => (
                      <div
                        key={customer.id}
                        className={`p-4 border rounded-lg cursor-pointer transition-colors ${selectedCustomerIds.includes(customer.id)
                          ? 'border-blue-500 bg-blue-50'
                          : 'border-gray-200 hover:border-gray-300'
                          }`}
                        onClick={() => handleToggleCustomer(customer.id)}
                      >
                        <div className="flex items-start gap-3">
                          <input
                            type="checkbox"
                            checked={selectedCustomerIds.includes(customer.id)}
                            onChange={() => handleToggleCustomer(customer.id)}
                            className="mt-1 w-4 h-4 text-blue-600 rounded"
                            onClick={(e) => e.stopPropagation()}
                          />
                          <div className="flex-1">
                            <div className="flex items-center gap-2">
                              <span className="font-medium text-gray-900">
                                {customer.firstname} {customer.lastname}
                              </span>
                              <span className="text-gray-500">•</span>
                              <span className="text-gray-600">{customer.phone}</span>
                            </div>
                            {(customer.location || customer.prefered_product) && (
                              <div className="mt-1 flex items-center gap-3 text-sm text-gray-500">
                                {customer.location && (
                                  <span>{customer.location}</span>
                                )}
                                {customer.prefered_product && (
                                  <span>{customer.prefered_product}</span>
                                )}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Modal Footer */}
            <div className="p-6 border-t border-gray-200 bg-gray-50">
              <div className="flex items-center justify-between">
                <p className="text-gray-600">
                  {selectedCustomerIds.length} customer{selectedCustomerIds.length !== 1 ? 's' : ''} selected
                </p>
                <div className="flex gap-3">
                  <button
                    onClick={() => setShowCustomerModal(false)}
                    className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSendCampaign}
                    disabled={selectedCustomerIds.length === 0 || sendingCampaign}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {sendingCampaign ? (
                      <>
                        <Loader2 className="w-4 h-4 animate-spin" />
                        Sending...
                      </>
                    ) : (
                      <>
                        <Send className="w-4 h-4" />
                        Send to {selectedCustomerIds.length} Customer{selectedCustomerIds.length !== 1 ? 's' : ''}
                      </>
                    )}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}