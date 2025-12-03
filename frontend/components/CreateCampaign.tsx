'use client'
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '../lib/api';
import { ArrowLeft, Loader2, Eye, MessageSquare, Smartphone } from 'lucide-react';

export function CreateCampaign() {
  const router = useRouter();
  const [name, setName] = useState('');
  const [template, setTemplate] = useState('');
  const [channel, setChannel] = useState<'whatsapp' | 'sms'>('whatsapp');
  const [scheduledDate, setScheduledDate] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim() || !template.trim() || !channel) {
      setError('Please fill in all required fields');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const campaign = await api.createCampaign({
        name,
        template,
        channel,
        scheduledDate: scheduledDate || undefined
      });
      router.push(`/campaign/${campaign.id}`);
    } catch (err) {
      setError('Failed to create campaign. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const templateVariables = [
    { name: 'first_name', description: 'Customer\'s first name' },
    { name: 'last_name', description: 'Customer\'s last name' },
    { name: 'phone', description: 'Customer\'s phone number' },
    { name: 'location', description: 'Customer\'s location' },
    { name: 'prefered_product', description: 'Customer\'s preferred product' },
  ];

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-6">
        <button
          onClick={() => router.push('/')}
          className="flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-4 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          Back to campaigns
        </button>

        <h1 className="text-gray-900 mb-2">Create New Campaign</h1>
        <p className="text-gray-600">
          Set up a new messaging campaign with a personalized template
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Form */}
        <div className="lg:col-span-2">
          <form onSubmit={handleSubmit} className="bg-white rounded-lg border border-gray-200 p-6">
            {error && (
              <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
                <p className="text-red-900">{error}</p>
              </div>
            )}

            <div className="mb-6">
              <label htmlFor="name" className="block text-gray-700 mb-2">
                Campaign Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Summer Sale 2026"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900"
                disabled={loading}
              />
            </div>

            <div className="mb-6">
              <label className="block text-gray-700 mb-2">
                Channel <span className="text-red-500">*</span>
              </label>
              <div className="grid grid-cols-2 gap-3">
                <button
                  type="button"
                  onClick={() => setChannel('whatsapp')}
                  className={`px-4 py-3 border-2 rounded-lg transition-all flex items-center justify-center gap-2 ${channel === 'whatsapp'
                    ? 'border-blue-500 bg-blue-50 text-blue-700'
                    : 'border-gray-300 bg-white text-gray-700 hover:border-gray-400'
                    }`}
                  disabled={loading}
                >
                  <MessageSquare className="w-5 h-5" />
                  WhatsApp
                </button>
                <button
                  type="button"
                  onClick={() => setChannel('sms')}
                  className={`px-4 py-3 border-2 rounded-lg transition-all flex items-center justify-center gap-2 ${channel === 'sms'
                    ? 'border-blue-500 bg-blue-50 text-blue-700'
                    : 'border-gray-300 bg-white text-gray-700 hover:border-gray-400'
                    }`}
                  disabled={loading}
                >
                  <Smartphone className="w-5 h-5" />
                  SMS
                </button>
              </div>
            </div>

            <div className="mb-6">
              <label htmlFor="scheduledDate" className="block text-gray-700 mb-2">
                Scheduled Date (Optional)
              </label>
              <input
                type="datetime-local"
                id="scheduledDate"
                value={scheduledDate}
                onChange={(e) => setScheduledDate(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900"
                disabled={loading}
              />
              <p className="mt-2 text-gray-500">
                Leave empty to start campaign immediately
              </p>
            </div>

            <div className="mb-6">
              <label htmlFor="template" className="block text-gray-700 mb-2">
                Message Template <span className="text-red-500">*</span>
              </label>
              <textarea
                id="template"
                value={template}
                onChange={(e) => setTemplate(e.target.value)}
                placeholder="Hi {first_name}, we have {prefered_product} available in {location}!"
                rows={6}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all font-mono resize-none text-gray-900"
                disabled={loading}
              />
              <p className="mt-2 text-gray-500">
                Use curly braces for variables, e.g., {'{first_name}'}
              </p>
            </div>

            <div className="flex gap-3">
              <button
                type="submit"
                disabled={loading || !name.trim() || !template.trim() || !channel}
                className="flex-1 px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2"
              >
                {loading ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  'Create Campaign'
                )}
              </button>
            </div>
          </form>
        </div>

        {/* Template Variables Guide */}
        <div className="lg:col-span-1">
          <div className="bg-white rounded-lg border border-gray-200 p-6 sticky top-6">
            <h2 className="text-gray-900 mb-4">Available Variables</h2>
            <div className="space-y-4">
              {templateVariables.map((variable) => (
                <div key={variable.name} className="pb-4 border-b border-gray-200 last:border-0 last:pb-0">
                  <code className="text-blue-600 bg-blue-50 px-2 py-1 rounded">
                    {`{${variable.name}}`}
                  </code>
                  <p className="text-gray-600 mt-2">{variable.description}</p>
                </div>
              ))}
            </div>

            <div className="mt-6 p-4 bg-gray-50 rounded-lg border border-gray-200">
              <p className="text-gray-700">
                <strong>Example:</strong>
              </p>
              <code className="text-gray-600 block mt-2">
                Hi {'{first_name}'} {'{last_name}'}, we have {'{prefered_product}'} in stock!
              </code>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}