import { useState, useEffect } from 'react';
import {
  CreateFingerprintWindow,
  ListFingerprintWindows,
  DeleteFingerprintWindow,
  CreateTabInFingerprintWindow,
  GetSingletonInstance,
  NavigateInstanceBrowser,
  ListTabs,
} from '../wailsjs/go/main/App';
import { commands } from '../wailsjs/go/models';

// Supported countries for fingerprint configuration
const SUPPORTED_COUNTRIES = [
  { code: 'US', name: 'United States', flag: '🇺🇸' },
  { code: 'GB', name: 'United Kingdom', flag: '🇬🇧' },
  { code: 'DE', name: 'Germany', flag: '🇩🇪' },
  { code: 'FR', name: 'France', flag: '🇫🇷' },
  { code: 'JP', name: 'Japan', flag: '🇯🇵' },
  { code: 'CN', name: 'China', flag: '🇨🇳' },
  { code: 'CA', name: 'Canada', flag: '🇨🇦' },
  { code: 'AU', name: 'Australia', flag: '🇦🇺' },
  { code: 'BR', name: 'Brazil', flag: '🇧🇷' },
  { code: 'IN', name: 'India', flag: '🇮🇳' },
];

interface FingerprintWithTabs {
  window: commands.FingerprintWindow;
  tabs: commands.TabInfo[];
}

type FingerprintManagerPageProps = {
  onGoToTabs?: () => void; // Reserved for future navigation needs
};

export function FingerprintManagerPage({ onGoToTabs: _onGoToTabs }: FingerprintManagerPageProps) {
  const [fingerprints, setFingerprints] = useState<FingerprintWithTabs[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [selectedCountry, setSelectedCountry] = useState<string>('US');
  const [newWindowUrl, setNewWindowUrl] = useState('');
  const [deletingWindow, setDeletingWindow] = useState<string | null>(null);
  const [runningInstance, setRunningInstance] = useState<commands.BrowserInstance | null>(null);
  const [expandedWindowId, setExpandedWindowId] = useState<string | null>(null);
  const [navigateUrl, setNavigateUrl] = useState('');
  const [showNavigate, setShowNavigate] = useState(false);
  const [selectedWindowId, setSelectedWindowId] = useState<string | null>(null);

  useEffect(() => {
    loadAll();
    const interval = setInterval(loadAll, 5000);
    return () => clearInterval(interval);
  }, []);

  async function loadAll() {
    try {
      setError(null);

      // Get singleton instance
      let inst: commands.BrowserInstance | null = null;
      try {
        inst = await GetSingletonInstance();
      } catch {
        // No singleton running, that's ok
      }
      setRunningInstance(inst);

      if (!inst) {
        setFingerprints([]);
        return;
      }

      // Get fingerprint windows
      const windows = await ListFingerprintWindows(inst.id, false);
      
      // Filter only top-level windows (not tabs), and get tabs for each
      const windowsOnly = (windows || []).filter(w => w.window_type === 'window');
      
      // Get tabs for the instance to associate with windows
      const allTabs = await ListTabs(inst.id);
      
      const fingerprintsWithTabs: FingerprintWithTabs[] = windowsOnly.map(win => {
        // Find tabs that belong to this window's context
        const windowTabs = (allTabs || []).filter(tab => 
          tab.ContextID === win.context_id || 
          tab.FingerprintSeed === win.seed
        );
        return {
          window: win,
          tabs: windowTabs,
        };
      });

      // Sort by creation time (newest first)
      fingerprintsWithTabs.sort((a, b) => 
        new Date(b.window.created_at).getTime() - new Date(a.window.created_at).getTime()
      );

      setFingerprints(fingerprintsWithTabs);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleCreateFingerprintWindow() {
    if (!runningInstance || creating) return;

    try {
      setCreating(true);
      setError(null);

      // Create a new fingerprint window with the selected country
      const newWindow = await CreateFingerprintWindow(runningInstance.id, selectedCountry, '');

      // If a URL is specified, create a tab in the window
      const targetUrl = newWindowUrl.trim() || 'about:blank';
      if (targetUrl !== 'about:blank' || targetUrl === 'about:blank') {
        // Always create at least one tab to show in browser
        await CreateTabInFingerprintWindow(newWindow.id, targetUrl);
      }

      // Open the browser window by navigating to the window
      if (runningInstance) {
        // Navigate to open the browser window
        await NavigateInstanceBrowser(runningInstance.id, targetUrl);
      }

      setShowCreate(false);
      setNewWindowUrl('');
      await loadAll();
      
      // Expand the newly created window
      setExpandedWindowId(newWindow.id);
    } catch (err) {
      setError(String(err));
    } finally {
      setCreating(false);
    }
  }

  async function deleteWindow(windowId: string) {
    if (!runningInstance || deletingWindow === windowId) return;

    try {
      setDeletingWindow(windowId);
      setError(null);
      await DeleteFingerprintWindow(windowId);
      
      // Clear expanded state if we deleted the expanded window
      if (expandedWindowId === windowId) {
        setExpandedWindowId(null);
      }
      
      await loadAll();
    } catch (err) {
      setError(String(err));
    } finally {
      setDeletingWindow(null);
    }
  }

  async function handleNavigate(_windowId: string, url: string) {
    if (!runningInstance) return;
    try {
      setError(null);
      // Navigate the browser to the URL - this will open/show the window
      await NavigateInstanceBrowser(runningInstance.id, url);
      await loadAll();
    } catch (err) {
      setError(String(err));
    }
  }

  async function handleNewTab(windowId: string) {
    if (!runningInstance) return;
    try {
      setError(null);
      const url = prompt('Enter URL for new tab:', 'about:blank');
      if (url !== null) {
        await CreateTabInFingerprintWindow(windowId, url);
        await loadAll();
      }
    } catch (err) {
      setError(String(err));
    }
  }

  function toggleExpand(windowId: string) {
    setExpandedWindowId(prev => prev === windowId ? null : windowId);
  }

  function openNavigateModal(windowId: string) {
    const fp = fingerprints.find(f => f.window.id === windowId);
    if (fp) {
      setSelectedWindowId(windowId);
      setNavigateUrl(fp.window.url || '');
      setShowNavigate(true);
    }
  }

  async function submitNavigate() {
    if (!selectedWindowId || !navigateUrl) return;
    try {
      setError(null);
      // Update URL via tab creation or direct navigation
      await handleNavigate(selectedWindowId, navigateUrl);
      setShowNavigate(false);
      setNavigateUrl('');
      setSelectedWindowId(null);
    } catch (err) {
      setError(String(err));
    }
  }

  function getCountryInfo(code: string): { name: string; flag: string } {
    const country = SUPPORTED_COUNTRIES.find(c => c.code === code);
    return country ? { name: country.name, flag: country.flag } : { name: code, flag: '🏳️' };
  }

  function truncateSeed(seed: string | undefined): string {
    if (!seed) return '-';
    return seed.slice(0, 12) + '...';
  }

  function formatDate(date: Date | string | undefined): string {
    if (!date) return '-';
    try {
      const d = new Date(date);
      return d.toLocaleString();
    } catch {
      return '-';
    }
  }

  function truncateUrl(url: string | undefined, maxLength: number = 40): string {
    if (!url || url === 'about:blank') return 'about:blank';
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + '...';
  }

  function getStatusColor(status: string): string {
    switch (status) {
      case 'active': return '#22c55e';
      case 'closed': return '#94a3b8';
      case 'error': return '#ef4444';
      default: return '#94a3b8';
    }
  }

  if (loading && fingerprints.length === 0) {
    return <div className="loading">Loading fingerprints...</div>;
  }

  return (
    <div className="fingerprint-manager-page">
      <div className="page-header">
        <h2>🔐 Fingerprint Manager</h2>
        <button
          className="btn-primary"
          onClick={() => setShowCreate(true)}
          disabled={!runningInstance}
        >
          + 新建指纹窗口
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {!runningInstance && (
        <div className="warning-banner">
          No browser instance running. Start an instance to manage fingerprints.
        </div>
      )}

      {/* Create New Fingerprint Modal */}
      {showCreate && (
        <div className="modal-overlay" onClick={() => setShowCreate(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h3>新建指纹窗口</h3>
            <div className="form-group">
              <label>国家/地区</label>
              <select
                value={selectedCountry}
                disabled={creating}
                onChange={e => setSelectedCountry(e.target.value)}
              >
                {SUPPORTED_COUNTRIES.map(c => (
                  <option key={c.code} value={c.code}>
                    {c.flag} {c.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="form-group">
              <label>起始 URL (可选)</label>
              <input
                type="text"
                value={newWindowUrl}
                disabled={creating}
                onChange={e => setNewWindowUrl(e.target.value)}
                placeholder="about:blank"
                onKeyDown={e => e.key === 'Enter' && handleCreateFingerprintWindow()}
              />
            </div>
            <div className="modal-actions">
              <button disabled={creating} onClick={() => setShowCreate(false)}>取消</button>
              <button className="btn-primary" onClick={handleCreateFingerprintWindow} disabled={creating}>
                {creating ? '创建中...' : '创建并打开浏览器'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Navigate Modal */}
      {showNavigate && (
        <div className="modal-overlay" onClick={() => setShowNavigate(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h3>导航到</h3>
            <div className="form-group">
              <label>URL</label>
              <input
                type="text"
                value={navigateUrl}
                onChange={e => setNavigateUrl(e.target.value)}
                placeholder="https://example.com"
                onKeyDown={e => e.key === 'Enter' && submitNavigate()}
                autoFocus
              />
            </div>
            <div className="modal-actions">
              <button onClick={() => setShowNavigate(false)}>取消</button>
              <button className="btn-primary" onClick={submitNavigate} disabled={!navigateUrl}>
                导航
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Fingerprint List */}
      {fingerprints.length === 0 ? (
        <div className="empty-state">
          <p>暂无指纹窗口。创建一个新的指纹窗口开始使用。</p>
          <button
            className="btn-primary"
            onClick={() => setShowCreate(true)}
            disabled={!runningInstance}
          >
            + 新建指纹窗口
          </button>
        </div>
      ) : (
        <div className="fingerprint-list">
          {fingerprints.map(({ window: fp, tabs }) => {
            const countryInfo = getCountryInfo(fp.country || 'US');
            const isExpanded = expandedWindowId === fp.id;
            
            return (
              <div key={fp.id} className={`fingerprint-card ${isExpanded ? 'expanded' : ''}`}>
                {/* Main Row - Fingerprint Info */}
                <div 
                  className="fingerprint-row"
                  onClick={() => toggleExpand(fp.id)}
                  style={{ cursor: 'pointer' }}
                >
                  <div className="fingerprint-country">
                    <span className="flag">{countryInfo.flag}</span>
                    <span className="country-name">{countryInfo.name}</span>
                  </div>
                  
                  <div className="fingerprint-info">
                    <div className="seed-row">
                      <span className="label">Seed:</span>
                      <span className="value">{truncateSeed(fp.seed)}</span>
                    </div>
                    <div className="url-row">
                      <span className="label">URL:</span>
                      <span className="value">{truncateUrl(fp.url)}</span>
                    </div>
                  </div>
                  
                  <div className="fingerprint-meta">
                    <div className="tab-count">
                      <span className="count">{tabs.length}</span>
                      <span className="label">标签页</span>
                    </div>
                    <div className="status">
                      <span 
                        className="status-dot"
                        style={{ backgroundColor: getStatusColor(fp.status) }}
                      />
                      <span className="status-text">{fp.status}</span>
                    </div>
                  </div>
                  
                  <div className="fingerprint-actions">
                    <button
                      className="btn-icon"
                      onClick={(e) => {
                        e.stopPropagation();
                        openNavigateModal(fp.id);
                      }}
                      title="导航"
                    >
                      ➡
                    </button>
                    <button
                      className="btn-icon"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleNewTab(fp.id);
                      }}
                      title="新建标签页"
                    >
                      +
                    </button>
                    <button
                      className="btn-icon btn-danger"
                      onClick={(e) => {
                        e.stopPropagation();
                        deleteWindow(fp.id);
                      }}
                      disabled={deletingWindow === fp.id}
                      title="删除"
                    >
                      {deletingWindow === fp.id ? '...' : '×'}
                    </button>
                    <span className={`expand-arrow ${isExpanded ? 'expanded' : ''}`}>
                      ▼
                    </span>
                  </div>
                </div>

                {/* Expanded Tabs Section */}
                {isExpanded && (
                  <div className="fingerprint-tabs-section">
                    <div className="tabs-header">
                      <span>标签页列表 ({tabs.length})</span>
                      <button
                        className="btn-secondary btn-small"
                        onClick={() => handleNewTab(fp.id)}
                      >
                        + 新建标签页
                      </button>
                    </div>
                    
                    {tabs.length === 0 ? (
                      <div className="no-tabs">
                        <p>暂无标签页</p>
                        <button
                          className="btn-primary btn-small"
                          onClick={() => handleNewTab(fp.id)}
                        >
                          + 新建标签页
                        </button>
                      </div>
                    ) : (
                      <div className="tabs-list">
                        {tabs.map(tab => (
                          <div key={tab.ID} className="tab-item">
                            <div className="tab-info">
                              <div className="tab-title">
                                {tab.Title || 'Untitled'}
                              </div>
                              <div className="tab-url">
                                {truncateUrl(tab.URL, 60)}
                              </div>
                              <div className="tab-meta">
                                <span>创建: {formatDate(tab.CreatedAt)}</span>
                                <span>活跃: {formatDate(tab.LastActiveAt)}</span>
                              </div>
                            </div>
                            <div className="tab-actions">
                              <button
                                className="btn-secondary btn-small"
                                onClick={() => openNavigateModal(fp.id)}
                                title="导航到此URL"
                              >
                                导航
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                    
                    <div className="fingerprint-details">
                      <div className="detail-row">
                        <span className="detail-label">Window ID:</span>
                        <span className="detail-value">{fp.id.slice(0, 16)}...</span>
                      </div>
                      <div className="detail-row">
                        <span className="detail-label">Context ID:</span>
                        <span className="detail-value">{fp.context_id?.slice(0, 16) || '-'}...</span>
                      </div>
                      <div className="detail-row">
                        <span className="detail-label">创建时间:</span>
                        <span className="detail-value">{formatDate(fp.created_at)}</span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      <style>{`
        .fingerprint-manager-page {
          padding: 16px;
          height: 100%;
          overflow-y: auto;
        }
        
        .page-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 16px;
        }
        
        .page-header h2 {
          margin: 0;
          font-size: 20px;
        }
        
        .error-banner {
          background: #fef2f2;
          border: 1px solid #fecaca;
          color: #dc2626;
          padding: 12px;
          border-radius: 6px;
          margin-bottom: 16px;
        }
        
        .warning-banner {
          background: #fef9c3;
          border: 1px solid #fef08a;
          color: #ca8a04;
          padding: 12px;
          border-radius: 6px;
          margin-bottom: 16px;
        }
        
        .modal-overlay {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(0, 0, 0, 0.5);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 1000;
        }
        
        .modal {
          background: white;
          padding: 24px;
          border-radius: 8px;
          min-width: 400px;
          max-width: 90%;
        }
        
        .modal h3 {
          margin: 0 0 16px 0;
        }
        
        .form-group {
          margin-bottom: 16px;
        }
        
        .form-group label {
          display: block;
          margin-bottom: 4px;
          font-weight: 500;
          color: #374151;
        }
        
        .form-group select,
        .form-group input {
          width: 100%;
          padding: 10px 12px;
          border: 1px solid #e5e7eb;
          border-radius: 6px;
          font-size: 14px;
        }
        
        .modal-actions {
          display: flex;
          justify-content: flex-end;
          gap: 8px;
          margin-top: 16px;
        }
        
        .btn-primary {
          background: #3b82f6;
          color: white;
          border: none;
          padding: 10px 16px;
          border-radius: 6px;
          cursor: pointer;
          font-weight: 500;
        }
        
        .btn-primary:disabled {
          background: #93c5fd;
          cursor: not-allowed;
        }
        
        .btn-secondary {
          background: #f3f4f6;
          color: #374151;
          border: 1px solid #e5e7eb;
          padding: 6px 12px;
          border-radius: 6px;
          cursor: pointer;
        }
        
        .btn-secondary:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
        
        .btn-small {
          padding: 4px 8px;
          font-size: 12px;
        }
        
        .empty-state {
          text-align: center;
          padding: 48px;
          color: #666;
        }
        
        .fingerprint-list {
          display: flex;
          flex-direction: column;
          gap: 12px;
        }
        
        .fingerprint-card {
          border: 1px solid #e5e7eb;
          border-radius: 8px;
          overflow: hidden;
          background: white;
          transition: box-shadow 0.2s;
        }
        
        .fingerprint-card:hover {
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        }
        
        .fingerprint-card.expanded {
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        }
        
        .fingerprint-row {
          display: flex;
          align-items: center;
          padding: 12px 16px;
          gap: 16px;
        }
        
        .fingerprint-country {
          display: flex;
          align-items: center;
          gap: 8px;
          min-width: 150px;
        }
        
        .fingerprint-country .flag {
          font-size: 24px;
        }
        
        .fingerprint-country .country-name {
          font-weight: 500;
          color: #1976d2;
        }
        
        .fingerprint-info {
          flex: 1;
          display: flex;
          flex-direction: column;
          gap: 4px;
        }
        
        .seed-row, .url-row {
          display: flex;
          gap: 8px;
          font-size: 13px;
        }
        
        .seed-row .label, .url-row .label {
          color: #888;
          min-width: 40px;
        }
        
        .seed-row .value {
          font-family: monospace;
          color: #666;
        }
        
        .url-row .value {
          font-family: monospace;
          color: #666;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
          max-width: 300px;
        }
        
        .fingerprint-meta {
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 8px;
          min-width: 80px;
        }
        
        .tab-count {
          display: flex;
          flex-direction: column;
          align-items: center;
        }
        
        .tab-count .count {
          font-size: 20px;
          font-weight: 600;
          color: #3b82f6;
        }
        
        .tab-count .label {
          font-size: 11px;
          color: #888;
        }
        
        .status {
          display: flex;
          align-items: center;
          gap: 4px;
        }
        
        .status-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }
        
        .status-text {
          font-size: 12px;
          color: #666;
        }
        
        .fingerprint-actions {
          display: flex;
          align-items: center;
          gap: 4px;
        }
        
        .btn-icon {
          background: transparent;
          border: 1px solid #e5e7eb;
          width: 32px;
          height: 32px;
          border-radius: 6px;
          cursor: pointer;
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 14px;
          color: #666;
        }
        
        .btn-icon:hover {
          background: #f3f4f6;
        }
        
        .btn-icon.btn-danger {
          color: #ef4444;
          border-color: #fecaca;
        }
        
        .btn-icon.btn-danger:hover {
          background: #fef2f2;
        }
        
        .btn-icon:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
        
        .expand-arrow {
          font-size: 10px;
          color: #888;
          margin-left: 8px;
          transition: transform 0.2s;
        }
        
        .expand-arrow.expanded {
          transform: rotate(180deg);
        }
        
        .fingerprint-tabs-section {
          border-top: 1px solid #e5e7eb;
          background: #f9fafb;
          padding: 16px;
        }
        
        .tabs-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 12px;
          font-weight: 500;
          color: #374151;
        }
        
        .no-tabs {
          text-align: center;
          padding: 24px;
          color: #888;
        }
        
        .no-tabs p {
          margin: 0 0 12px 0;
        }
        
        .tabs-list {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }
        
        .tab-item {
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
          padding: 12px;
          background: white;
          border: 1px solid #e5e7eb;
          border-radius: 6px;
        }
        
        .tab-info {
          flex: 1;
        }
        
        .tab-title {
          font-weight: 500;
          color: #333;
          margin-bottom: 4px;
        }
        
        .tab-url {
          font-family: monospace;
          font-size: 12px;
          color: #666;
          margin-bottom: 4px;
        }
        
        .tab-meta {
          font-size: 11px;
          color: #888;
          display: flex;
          gap: 16px;
        }
        
        .tab-actions {
          display: flex;
          gap: 4px;
        }
        
        .fingerprint-details {
          margin-top: 16px;
          padding-top: 16px;
          border-top: 1px dashed #e5e7eb;
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 8px;
        }
        
        .detail-row {
          display: flex;
          flex-direction: column;
          gap: 2px;
        }
        
        .detail-label {
          font-size: 11px;
          color: #888;
        }
        
        .detail-value {
          font-family: monospace;
          font-size: 12px;
          color: #666;
        }
        
        .loading {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 200px;
          color: #888;
        }
      `}</style>
    </div>
  );
}
