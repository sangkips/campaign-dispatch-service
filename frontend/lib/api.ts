// API client for Campaign Dispatch Service backend
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Frontend Campaign interface
export interface Campaign {
  id: string;
  name: string;
  status: 'draft' | 'scheduled' | 'sending' | 'sent' | 'failed';
  template: string;
  channel: 'whatsapp' | 'sms';
  scheduledDate?: string;
  createdAt: string;
  totalMessages: number;
  sentMessages: number;
  deliveredMessages: number;
  failedMessages: number;
}

export interface PersonalizedPreview {
  customerId: string;
  customerName: string;
  personalizedMessage: string;
}

export interface Customer {
  id: number;
  phone: string;
  firstname: string;
  lastname: string;
  location?: string;
  prefered_product?: string;
  created_at: string;
}

// Backend API response interfaces
interface BackendCampaign {
  id: number;
  name: string;
  channel: string;
  status: string;
  base_template: string;
  scheduled_at?: string;
  created_at: string;
}

interface BackendCampaignWithStats extends BackendCampaign {
  stats: {
    total: number;
    pending: number;
    sending: number;
    sent: number;
    failed: number;
  };
}

interface BackendListResponse {
  data: BackendCampaignWithStats[];
  pagination: {
    page: number;
    page_size: number;
    total_count: number;
    total_pages: number;
  };
}

interface BackendPersonalizedPreview {
  rendered_message: string;
  used_template: string;
  customer: {
    id: number;
    first_name: string;
    last_name: string;
    phone: string;
    location?: string;
    prefered_product?: string;
  };
}

// Map backend status to frontend status (direct mapping, no transformation needed)
const mapStatus = (status: string): Campaign['status'] => {
  const validStatuses: Campaign['status'][] = ['draft', 'scheduled', 'sending', 'sent', 'failed'];
  return validStatuses.includes(status as Campaign['status'])
    ? (status as Campaign['status'])
    : 'draft';
};

// Convert backend campaign to frontend format
const mapCampaign = (backendCampaign: BackendCampaign, stats?: BackendCampaignWithStats['stats']): Campaign => {
  return {
    id: String(backendCampaign.id),
    name: backendCampaign.name,
    status: mapStatus(backendCampaign.status),
    template: backendCampaign.base_template,
    channel: backendCampaign.channel as 'whatsapp' | 'sms',
    scheduledDate: backendCampaign.scheduled_at,
    createdAt: backendCampaign.created_at,
    totalMessages: stats?.total || 0,
    sentMessages: stats?.sent || 0,
    deliveredMessages: stats?.sent || 0, // Backend doesn't track delivered separately
    failedMessages: stats?.failed || 0,
  };
};

export const api = {
  // GET /campaigns?page=1&page_size=10&status=active&channel=whatsapp
  getCampaigns: async (page = 1, limit = 10, status?: string, channel?: string): Promise<{ campaigns: Campaign[], total: number }> => {
    const params = new URLSearchParams({
      page: String(page),
      page_size: String(limit),
    });

    if (status && status !== 'all') {
      params.append('status', status);
    }

    if (channel && channel !== 'all') {
      params.append('channel', channel);
    }

    const response = await fetch(`${API_BASE_URL}/campaigns?${params}`);
    if (!response.ok) {
      throw new Error('Failed to fetch campaigns');
    }

    const data: BackendListResponse = await response.json();

    return {
      campaigns: data.data.map(c => mapCampaign(c, c.stats)),
      total: data.pagination.total_count,
    };
  },

  // GET /campaigns/:id
  getCampaign: async (id: string): Promise<Campaign | null> => {
    const response = await fetch(`${API_BASE_URL}/campaigns/${id}`);
    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error('Failed to fetch campaign');
    }

    const data: BackendCampaignWithStats = await response.json();
    return mapCampaign(data, data.stats);
  },

  // POST /campaigns
  createCampaign: async (data: {
    name: string;
    template: string;
    channel: 'whatsapp' | 'sms';
    scheduledDate?: string;
  }): Promise<Campaign> => {
    // Convert datetime-local format (YYYY-MM-DDTHH:mm) to ISO 8601 format
    let scheduledAt: string | undefined;
    if (data.scheduledDate) {
      // datetime-local gives us "2025-12-04T13:34", we need to convert to ISO 8601
      const date = new Date(data.scheduledDate);
      scheduledAt = date.toISOString(); // This gives us "2025-12-04T10:34:00.000Z"
    }

    const requestBody = {
      name: data.name,
      base_template: data.template,
      channel: data.channel,
      scheduled_at: scheduledAt,
    };

    const response = await fetch(`${API_BASE_URL}/campaigns`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(requestBody),
    });

    if (!response.ok) {
      throw new Error('Failed to create campaign');
    }

    const backendCampaign: BackendCampaign = await response.json();
    return mapCampaign(backendCampaign);
  },

  // POST /campaigns/:id/send
  sendCampaign: async (id: string, customerIds: number[]): Promise<{
    campaign_id: number;
    messages_queued: number;
    status: string;
  }> => {
    const response = await fetch(`${API_BASE_URL}/campaigns/${id}/send`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        customer_ids: customerIds,
      }),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || 'Failed to send campaign');
    }

    return await response.json();
  },

  // GET /customers?page=1&limit=100
  getCustomers: async (page = 1, limit = 100): Promise<{ customers: Customer[], total: number }> => {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String((page - 1) * limit),
    });

    const response = await fetch(`${API_BASE_URL}/customers?${params}`);
    if (!response.ok) {
      throw new Error('Failed to fetch customers');
    }

    const customers: Customer[] = await response.json();

    return {
      customers,
      total: customers.length, // Backend doesn't return total count yet
    };
  },

};