import { create } from "zustand";

export interface OverviewStats {
	total: number;
	active: number;
	offered: number;
	success_rate: number;
	this_week_new: number;
	last_updated: string;
}

export interface FunnelStage {
	status: string;
	label: string;
	count: number;
	percentage: number;
	avg_days: number;
}

export interface FunnelStats {
	stages: FunnelStage[];
	conversion: number;
	total_time_avg: number;
}

export interface SourceStats {
	source: string;
	count: number;
	percentage: number;
	applied: number;
	interviewed: number;
	offered: number;
	success_rate: number;
}

export interface SourceAnalysis {
	sources: SourceStats[];
	top_source: string;
	best_source: string;
}

export interface TimePoint {
	date: string;
	label: string;
	new: number;
	moved: number;
	total: number;
}

export interface TrendStats {
	period: string;
	points: TimePoint[];
	growth_rate: number;
	is_growing: boolean;
}

export interface InsightItem {
	type: string;
	severity: "info" | "warning" | "critical" | "success";
	title: string;
	message: string;
	action: string;
	related_lead?: string;
}

export interface InsightStats {
	total: number;
	urgent: number;
	tips: number;
	insights: InsightItem[];
	generated_at: string;
}

export interface SummaryStats {
	overview: OverviewStats;
	funnel: FunnelStats;
	sources: SourceAnalysis;
	trends: TrendStats;
	insights: InsightStats;
	generated_at: string;
}

interface StatsState {
	overview: OverviewStats | null;
	funnel: FunnelStats | null;
	sources: SourceAnalysis | null;
	trends: TrendStats | null;
	insights: InsightStats | null;
	summary: SummaryStats | null;
	isLoading: boolean;
	error: string | null;

	// Actions
	fetchOverview: () => Promise<void>;
	fetchFunnel: () => Promise<void>;
	fetchSources: () => Promise<void>;
	fetchTrends: (period?: string) => Promise<void>;
	fetchInsights: () => Promise<void>;
	fetchSummary: () => Promise<void>;
	fetchAll: () => Promise<void>;
	clearError: () => void;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export const useStatsStore = create<StatsState>((set, get) => ({
	overview: null,
	funnel: null,
	sources: null,
	trends: null,
	insights: null,
	summary: null,
	isLoading: false,
	error: null,

	fetchOverview: async () => {
		set({ isLoading: true, error: null });
		try {
			const res = await fetch(`${API_BASE}/api/stats/overview`);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const json = await res.json();
			set({ overview: json.data, isLoading: false });
		} catch (err) {
			set({ error: err instanceof Error ? err.message : "Unknown error", isLoading: false });
		}
	},

	fetchFunnel: async () => {
		set({ isLoading: true, error: null });
		try {
			const res = await fetch(`${API_BASE}/api/stats/funnel`);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const json = await res.json();
			set({ funnel: json.data, isLoading: false });
		} catch (err) {
			set({ error: err instanceof Error ? err.message : "Unknown error", isLoading: false });
		}
	},

	fetchSources: async () => {
		set({ isLoading: true, error: null });
		try {
			const res = await fetch(`${API_BASE}/api/stats/sources`);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const json = await res.json();
			set({ sources: json.data, isLoading: false });
		} catch (err) {
			set({ error: err instanceof Error ? err.message : "Unknown error", isLoading: false });
		}
	},

	fetchTrends: async (period = "month") => {
		set({ isLoading: true, error: null });
		try {
			const res = await fetch(`${API_BASE}/api/stats/trends?period=${period}`);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const json = await res.json();
			set({ trends: json.data, isLoading: false });
		} catch (err) {
			set({ error: err instanceof Error ? err.message : "Unknown error", isLoading: false });
		}
	},

	fetchInsights: async () => {
		set({ isLoading: true, error: null });
		try {
			const res = await fetch(`${API_BASE}/api/stats/insights`);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const json = await res.json();
			set({ insights: json.data, isLoading: false });
		} catch (err) {
			set({ error: err instanceof Error ? err.message : "Unknown error", isLoading: false });
		}
	},

	fetchSummary: async () => {
		set({ isLoading: true, error: null });
		try {
			const res = await fetch(`${API_BASE}/api/stats/summary`);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			const json = await res.json();
			set({ summary: json.data, isLoading: false });
		} catch (err) {
			set({ error: err instanceof Error ? err.message : "Unknown error", isLoading: false });
		}
	},

	fetchAll: async () => {
		const { fetchOverview, fetchFunnel, fetchSources, fetchInsights } = get();
		await Promise.all([
			fetchOverview(),
			fetchFunnel(),
			fetchSources(),
			fetchInsights(),
		]);
	},

	clearError: () => set({ error: null }),
}));
