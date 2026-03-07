/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Agent cloud log viewer component.
 *
 * Fetches and displays structured log entries from the hub's Cloud Logging
 * endpoint. Supports manual refresh, severity-colored rows, expandable JSON
 * detail, and live SSE streaming.
 */

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { apiFetch } from '../../client/api.js';
import './json-browser.js';

interface CloudLogEntry {
  timestamp: string;
  severity: string;
  message: string;
  labels?: Record<string, string>;
  resource?: Record<string, unknown>;
  jsonPayload?: Record<string, unknown>;
  insertId: string;
  sourceLocation?: { file?: string; line?: string; function?: string };
}

interface CloudLogsResponse {
  entries: CloudLogEntry[];
  nextPageToken?: string;
  hasMore?: boolean;
}

const MAX_BUFFER = 2000;

@customElement('scion-agent-log-viewer')
export class ScionAgentLogViewer extends LitElement {
  @property()
  agentId = '';

  @state() private entries: CloudLogEntry[] = [];
  @state() private entryMap = new Map<string, CloudLogEntry>();
  @state() private loading = false;
  @state() private error: string | null = null;
  @state() private streaming = false;
  @state() private expandedIds = new Set<string>();
  @state() private loaded = false;

  private eventSource: EventSource | null = null;

  static override styles = css`
    :host {
      display: block;
    }

    /* Toolbar */
    .toolbar {
      display: flex;
      align-items: center;
      justify-content: flex-end;
      gap: 0.75rem;
      margin-bottom: 1rem;
    }
    .toolbar-label {
      font-size: 0.8125rem;
      color: var(--scion-text-muted, #64748b);
      margin-right: 0.25rem;
    }

    /* Log table */
    .log-table {
      width: 100%;
      font-family: var(--scion-font-mono, monospace);
      font-size: 0.8125rem;
      border-collapse: collapse;
    }

    .log-row {
      cursor: pointer;
      border-bottom: 1px solid var(--scion-border, #e2e8f0);
      transition: background 0.1s ease;
    }
    .log-row:hover {
      background: var(--scion-bg-subtle, #f1f5f9);
    }

    .log-row td {
      padding: 0.375rem 0.5rem;
      vertical-align: top;
      white-space: nowrap;
    }
    .log-row td.msg {
      white-space: normal;
      word-break: break-word;
      max-width: 0;
      width: 100%;
    }

    /* Expanded detail row */
    .detail-row td {
      padding: 0.75rem 0.5rem 0.75rem 2.5rem;
      border-bottom: 1px solid var(--scion-border, #e2e8f0);
      background: var(--scion-bg-subtle, #f1f5f9);
    }

    /* Severity badges */
    .sev {
      display: inline-block;
      padding: 0.0625rem 0.375rem;
      border-radius: 0.25rem;
      font-size: 0.6875rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.03em;
      line-height: 1.4;
    }
    .sev-DEBUG { background: var(--scion-neutral-100, #f1f5f9); color: var(--scion-neutral-500, #64748b); }
    .sev-INFO { background: var(--scion-primary-50, #eff6ff); color: var(--scion-primary-700, #1d4ed8); }
    .sev-WARNING { background: var(--scion-warning-50, #fffbeb); color: var(--scion-warning-700, #b45309); }
    .sev-ERROR { background: var(--scion-danger-50, #fef2f2); color: var(--scion-danger-700, #b91c1c); }
    .sev-CRITICAL { background: var(--scion-danger-100, #fee2e2); color: var(--scion-danger-800, #991b1b); font-weight: 700; }

    /* Timestamp */
    .ts {
      color: var(--scion-text-muted, #64748b);
      font-size: 0.75rem;
    }

    /* Subsystem */
    .sub {
      color: var(--scion-text-secondary, #475569);
      font-size: 0.75rem;
    }

    /* Empty / Loading / Error */
    .state-msg {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 3rem 2rem;
      color: var(--scion-text-muted, #64748b);
      gap: 0.75rem;
    }
    .state-msg sl-spinner {
      font-size: 1.5rem;
    }
    .state-msg sl-icon {
      font-size: 2rem;
      opacity: 0.4;
    }

    /* Date divider */
    .date-divider {
      padding: 0.5rem 0.5rem 0.25rem;
      font-size: 0.6875rem;
      font-weight: 600;
      color: var(--scion-text-muted, #64748b);
      text-transform: uppercase;
      letter-spacing: 0.05em;
      border-bottom: 1px solid var(--scion-border, #e2e8f0);
      background: var(--scion-surface, #ffffff);
    }

    .stream-indicator {
      display: inline-flex;
      align-items: center;
      gap: 0.375rem;
      font-size: 0.75rem;
      color: var(--scion-success-600, #16a34a);
    }
    .stream-dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: var(--scion-success-500, #22c55e);
      animation: pulse 1.5s ease-in-out infinite;
    }
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.3; }
    }
  `;

  override connectedCallback(): void {
    super.connectedCallback();
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.stopStream();
  }

  /** Called by the parent when the logs tab is first shown. */
  loadLogs(): void {
    if (this.loaded) return;
    this.loaded = true;
    void this.fetchLogs();
  }

  private async fetchLogs(): Promise<void> {
    if (!this.agentId) return;
    this.loading = true;
    this.error = null;

    try {
      // If we have entries, fetch only newer ones
      let url = `/api/v1/agents/${this.agentId}/cloud-logs?tail=200`;
      if (this.entries.length > 0) {
        const newest = this.entries[0].timestamp;
        url = `/api/v1/agents/${this.agentId}/cloud-logs?since=${encodeURIComponent(newest)}&tail=200`;
      }

      const res = await apiFetch(url);
      if (!res.ok) {
        const errData = await res.json().catch(() => ({})) as { error?: { message?: string }; message?: string };
        throw new Error(
          (errData.error as { message?: string })?.message || errData.message || `HTTP ${res.status}`
        );
      }

      const data = (await res.json()) as CloudLogsResponse;
      this.mergeEntries(data.entries);
    } catch (err) {
      this.error = err instanceof Error ? err.message : 'Failed to fetch logs';
    } finally {
      this.loading = false;
    }
  }

  private mergeEntries(newEntries: CloudLogEntry[]): void {
    for (const entry of newEntries) {
      if (!this.entryMap.has(entry.insertId)) {
        this.entryMap.set(entry.insertId, entry);
      }
    }

    // Sort descending by timestamp (newest first)
    const sorted = Array.from(this.entryMap.values()).sort(
      (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    );

    // Cap buffer
    if (sorted.length > MAX_BUFFER) {
      const evicted = sorted.splice(MAX_BUFFER);
      for (const e of evicted) {
        this.entryMap.delete(e.insertId);
      }
    }

    this.entries = sorted;
  }

  private startStream(): void {
    if (this.eventSource) return;
    this.streaming = true;

    const url = `/api/v1/agents/${this.agentId}/cloud-logs/stream`;
    this.eventSource = new EventSource(url);

    this.eventSource.addEventListener('log', (event: Event) => {
      try {
        const entry = JSON.parse((event as MessageEvent).data) as CloudLogEntry;
        this.mergeEntries([entry]);
      } catch {
        // Skip unparseable entries
      }
    });

    this.eventSource.addEventListener('timeout', () => {
      // Server timed out, reconnect
      this.stopStream();
      this.startStream();
    });

    this.eventSource.onerror = () => {
      // EventSource will auto-reconnect for transient errors
    };
  }

  private stopStream(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    this.streaming = false;
  }

  private toggleExpand(insertId: string): void {
    if (this.expandedIds.has(insertId)) {
      this.expandedIds.delete(insertId);
    } else {
      this.expandedIds.add(insertId);
    }
    this.requestUpdate();
  }

  private handleStreamToggle(e: Event): void {
    const checked = (e.target as HTMLInputElement).checked;
    if (checked) {
      this.startStream();
    } else {
      this.stopStream();
    }
  }

  private handleRefresh(): void {
    void this.fetchLogs();
  }

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------

  override render() {
    return html`
      ${this.renderToolbar()}
      ${this.renderContent()}
    `;
  }

  private renderToolbar() {
    return html`
      <div class="toolbar">
        ${this.streaming
          ? html`<span class="stream-indicator"><span class="stream-dot"></span>Streaming</span>`
          : nothing}
        <sl-button
          size="small"
          variant="default"
          ?loading=${this.loading}
          ?disabled=${this.loading || this.streaming}
          @click=${this.handleRefresh}
        >
          <sl-icon slot="prefix" name="arrow-clockwise"></sl-icon>
          Refresh
        </sl-button>
        <span class="toolbar-label">Stream</span>
        <sl-switch
          size="small"
          ?checked=${this.streaming}
          @sl-change=${this.handleStreamToggle}
        ></sl-switch>
      </div>
    `;
  }

  private renderContent() {
    if (this.loading && this.entries.length === 0) {
      return html`
        <div class="state-msg">
          <sl-spinner></sl-spinner>
          <span>Loading logs...</span>
        </div>
      `;
    }

    if (this.error && this.entries.length === 0) {
      return html`
        <div class="state-msg">
          <sl-icon name="exclamation-triangle"></sl-icon>
          <span>${this.error}</span>
          <sl-button size="small" @click=${this.handleRefresh}>Retry</sl-button>
        </div>
      `;
    }

    if (this.entries.length === 0) {
      return html`
        <div class="state-msg">
          <sl-icon name="file-text"></sl-icon>
          <span>No log entries found</span>
        </div>
      `;
    }

    return html`<table class="log-table"><tbody>${this.renderEntries()}</tbody></table>`;
  }

  private renderEntries() {
    const rows: unknown[] = [];
    let lastDate = '';

    for (const entry of this.entries) {
      const d = new Date(entry.timestamp);
      const dateStr = d.toLocaleDateString('en', { year: 'numeric', month: 'short', day: 'numeric' });

      if (dateStr !== lastDate) {
        lastDate = dateStr;
        rows.push(html`
          <tr><td colspan="4" class="date-divider">${dateStr}</td></tr>
        `);
      }

      const isExpanded = this.expandedIds.has(entry.insertId);
      const timeStr = d.toLocaleTimeString('en', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit', fractionalSecondDigits: 3 } as Intl.DateTimeFormatOptions);
      const subsystem = entry.jsonPayload?.['subsystem'] as string
        || entry.labels?.['component']
        || '';

      rows.push(html`
        <tr class="log-row" @click=${() => this.toggleExpand(entry.insertId)}>
          <td class="ts">${timeStr}</td>
          <td><span class="sev sev-${entry.severity}">${entry.severity}</span></td>
          <td class="sub">${subsystem}</td>
          <td class="msg">${entry.message}</td>
        </tr>
      `);

      if (isExpanded) {
        rows.push(html`
          <tr class="detail-row">
            <td colspan="4">
              <scion-json-browser .data=${this.buildDetailObject(entry)} expand-first></scion-json-browser>
            </td>
          </tr>
        `);
      }
    }

    return rows;
  }

  private buildDetailObject(entry: CloudLogEntry): Record<string, unknown> {
    const obj: Record<string, unknown> = {
      timestamp: entry.timestamp,
      severity: entry.severity,
      message: entry.message,
    };
    if (entry.labels && Object.keys(entry.labels).length > 0) {
      obj['labels'] = entry.labels;
    }
    if (entry.jsonPayload && Object.keys(entry.jsonPayload).length > 0) {
      obj['jsonPayload'] = entry.jsonPayload;
    }
    if (entry.resource && Object.keys(entry.resource).length > 0) {
      obj['resource'] = entry.resource;
    }
    if (entry.sourceLocation) {
      obj['sourceLocation'] = entry.sourceLocation;
    }
    obj['insertId'] = entry.insertId;
    return obj;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'scion-agent-log-viewer': ScionAgentLogViewer;
  }
}
