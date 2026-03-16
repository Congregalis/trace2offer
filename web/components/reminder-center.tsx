"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Bell, BellRing, Mail, RefreshCw, Smartphone, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");

interface ReminderItem {
  id: string;
  lead_id: string;
  type: string;
  title: string;
  message: string;
  due_at: string;
  severity: string;
  methods: string[];
  company: string;
  position: string;
  next_action: string;
}

interface APIErrorPayload {
  message?: string;
  error?: string;
}

interface APIPayload {
  data?: ReminderItem[];
}

function formatDueAt(value: string): string {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return "-";
  }
  const date = new Date(timestamp);
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, "0")}:${String(
    date.getMinutes()
  ).padStart(2, "0")}`;
}

async function parseAPIError(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as APIErrorPayload;
    const message = [payload.message, payload.error].filter(Boolean).join(": ");
    if (message) {
      return message;
    }
  } catch {
    // ignore non-json
  }
  return `加载提醒失败 (HTTP ${response.status})`;
}

function methodLabel(method: string): string {
  switch (method) {
    case "in_app":
      return "页面内";
    case "email":
      return "邮件";
    case "web_push":
      return "Web Push";
    default:
      return method;
  }
}

export function ReminderCenter() {
  const [items, setItems] = useState<ReminderItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notificationPermission, setNotificationPermission] = useState<NotificationPermission | "unsupported">(
    typeof Notification === "undefined" ? "unsupported" : Notification.permission
  );

  const notifiedRef = useRef<Set<string>>(new Set());

  const fetchDueReminders = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/api/reminders/due`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!response.ok) {
        throw new Error(await parseAPIError(response));
      }

      const payload = (await response.json()) as APIPayload;
      const list = Array.isArray(payload.data) ? payload.data : [];
      setItems(list);

      for (const item of list) {
        if (notifiedRef.current.has(item.id)) {
          continue;
        }
        notifiedRef.current.add(item.id);

        if (item.methods.includes("in_app")) {
          toast.warning(item.title, { description: item.message });
        }
        if (
          item.methods.includes("web_push") &&
          typeof Notification !== "undefined" &&
          Notification.permission === "granted"
        ) {
          new Notification(item.title, {
            body: item.message,
            tag: item.id,
          });
        }
      }
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : "加载提醒失败";
      setError(message);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchDueReminders();

    const timer = window.setInterval(() => {
      void fetchDueReminders();
    }, 60_000);

    return () => {
      window.clearInterval(timer);
    };
  }, [fetchDueReminders]);

  const requestNotificationPermission = async () => {
    if (typeof Notification === "undefined") {
      return;
    }
    const permission = await Notification.requestPermission();
    setNotificationPermission(permission);
    if (permission === "granted") {
      toast.success("浏览器通知已开启");
      return;
    }
    toast.error("浏览器通知未授权");
  };

  return (
    <section className="rounded-lg border border-border bg-card p-4 space-y-3">
      <div className="flex items-center justify-between gap-3">
        <div className="inline-flex items-center gap-2 text-sm font-medium text-foreground">
          {items.length > 0 ? <BellRing className="h-4 w-4 text-warning" /> : <Bell className="h-4 w-4 text-muted-foreground" />}
          跟进提醒中心
          <span className="rounded bg-secondary px-2 py-0.5 text-xs text-muted-foreground">{items.length} 条待处理</span>
        </div>
        <div className="inline-flex items-center gap-2">
          {notificationPermission !== "granted" && notificationPermission !== "unsupported" ? (
            <Button size="sm" variant="outline" onClick={requestNotificationPermission}>
              <Smartphone className="mr-1 h-4 w-4" />
              开启系统通知
            </Button>
          ) : null}
          <Button size="sm" variant="outline" asChild>
            <a href={`${API_BASE_URL}/api/calendar/interviews.ics`} target="_blank" rel="noopener noreferrer">
              导出 ICS
            </a>
          </Button>
          <Button size="sm" variant="ghost" onClick={() => void fetchDueReminders()} disabled={isLoading}>
            <RefreshCw className={cn("h-4 w-4", isLoading ? "animate-spin" : "")} />
          </Button>
        </div>
      </div>

      {error ? (
        <p className="inline-flex items-center gap-1 text-xs text-destructive">
          <TriangleAlert className="h-3.5 w-3.5" />
          {error}
        </p>
      ) : null}

      {items.length === 0 ? (
        <p className="text-xs text-muted-foreground">当前没有到期提醒，继续保持这个节奏。</p>
      ) : (
        <div className="space-y-2">
          {items.slice(0, 5).map((item) => (
            <article key={item.id} className="rounded-md border border-border/80 bg-secondary/20 p-3 space-y-1.5">
              <div className="flex items-center justify-between gap-2">
                <p className="text-sm font-medium text-foreground">{item.title}</p>
                <span className="text-[11px] text-muted-foreground">{formatDueAt(item.due_at)}</span>
              </div>
              <p className="text-xs text-muted-foreground">{item.message}</p>
              <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                {item.methods.map((method) => (
                  <span key={`${item.id}:${method}`} className="inline-flex items-center gap-1 rounded bg-background px-1.5 py-0.5">
                    {method === "email" ? <Mail className="h-3 w-3" /> : <Bell className="h-3 w-3" />}
                    {methodLabel(method)}
                  </span>
                ))}
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}
