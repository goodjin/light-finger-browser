import { useState, useEffect } from 'react';
import {
  CreateFingerprintWindow,
  ListFingerprintWindows,
  DeleteFingerprintWindow,
  CreateTabInFingerprintWindow,
  GetSingletonInstance,
  NavigateInstanceBrowser,
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

interface FingerprintGroup {
  country: string;
  countryName: string;
  flag: string;
  windows: commands.FingerprintWindow[];
}

type FingerprintManagerPageProps = {
  onGoToTabs?: () => void;
};

export function FingerprintManagerPage({ onGoToTabs }: FingerprintManagerPageProps) {
  const [groups, setGroups] = useState<FingerprintGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [selectedCountry, setSelectedCountry] = useState<string>('US');
  const [newWindowUrl, setNewWindowUrl] = useState('about:blank');
  const [deletingGroup, setDeletingGroup] = useState<string | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<FingerprintGroup | null>(null);
  const [deletingWindow, setDeletingWindow] = useState<string | null>(null);
  const [runningInstance, setRunningInstance] = useState<commands.BrowserInstance | null>(null);

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
        setGroups([]);
        return;
      }

      // Get fingerprint windows and group by country
      const windows = await ListFingerprintWindows(inst.id, false);
      const countryMap = new Map<string, commands.FingerprintWindow[]>();

      for (const win of windows || []) {
        const country = win.country || 'US';
        if (!countryMap.has(country)) {
          countryMap.set(country, []);
        }
        countryMap.get(country)!.push(win);
      }

      // Convert to array sorted by country name
      const grouped: FingerprintGroup[] = [];
      countryMap.forEach((windows, country) => {
        const countryInfo = SUPPORTED_COUNTRIES.find(c => c.code === country);
        grouped.push({
          country,
          countryName: countryInfo?.name || country,
          flag: countryInfo?.flag || '🏳️',
          windows,
        });
      });

      // Sort by country name
      grouped.sort((a, b) => a.countryName.localeCompare(b.countryName));

      setGroups(grouped);
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
      const window = await CreateFingerprintWindow(runningInstance.id, selectedCountry, '');

      // If a URL is specified, create a tab in the window
      if (newWindowUrl && newWindowUrl !== 'about:blank') {
        await CreateTabInFingerprintWindow(window.id, newWindowUrl);
      }

      setShowCreate(false);
      setNewWindowUrl('about:blank');
      await loadAll();
    } catch (err) {
      setError(String(err));
    } finally {
      setCreating(false);
    }
  }

  async function deleteGroup(country: string) {
    if (!runningInstance || deletingGroup === country) return;

    try {
      setDeletingGroup(country);
      setError(null);

      // Close all windows in this country group
      const group = groups.find(g => g.country === country);
      if (group) {
        for (const win of group.windows) {
          try {
            await DeleteFingerprintWindow(win.id);
          } catch {
            // Window might already be closed, continue
          }
        }
      }

      setSelectedGroup(null);
      await loadAll();
    } catch (err) {
      setError(String(err));
    } finally {
      setDeletingGroup(null);
    }
  }

  async function closeWindow(windowId: string) {
    if (!runningInstance || deletingWindow === windowId) return;

    try {
      setDeletingWindow(windowId);
      setError(null);
      await DeleteFingerprintWindow(windowId);
      await loadAll();

      // Update selected group if still shown
      if (selectedGroup) {
        const updatedGroup = groups.find(g => g.country === selectedGroup.country);
        if (updatedGroup) {
          setSelectedGroup(updatedGroup);
        }
      }
    } catch (err) {
      setError(String(err));
    } finally {
      setDeletingWindow(null);
    }
  }

  function handleGroupClick(group: FingerprintGroup) {
    setSelectedGroup(group);
    if (onGoToTabs) {
      onGoToTabs();
    }
  }

  if (loading && groups.length === 0) {
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

      {showCreate && (
        <div className="modal-overlay">
          <div className="modal">
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
              />
            </div>
            <div className="modal-actions">
              <button disabled={creating} onClick={() => setShowCreate(false)}>取消</button>
              <button className="btn-primary" onClick={handleCreateFingerprintWindow} disabled={creating}>
                {creating ? '创建中...' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}

      {selectedGroup && (
        <div className="modal-overlay">
          <div className="modal modal-wide">
            <h3>
              {selectedGroup.flag} {selectedGroup.countryName} 指纹详情
            </h3>
            <p className="section-desc">
              该国家/地区共有 {selectedGroup.windows.length} 个指纹窗口
            </p>
            <div className="fingerprint-detail-list">
              {selectedGroup.windows.map(win => (
                <div key={win.id} className="fingerprint-detail-item">
                  <div className="fingerprint-detail-info">
                    <div className="fingerprint-seed">
                      Seed: {win.seed ? win.seed.slice(0, 16) + '...' : '-'}
                    </div>
                    <div className="fingerprint-url">
                      URL: {win.url || '-'}
                    </div>
                    <div className="fingerprint-meta">
                      Window ID: {win.id.slice(0, 12)}... | 
                      Type: {win.window_type} | 
                      Status: {win.status}
                    </div>
                    <div className="fingerprint-meta">
                      Created: {win.created_at ? new Date(win.created_at).toLocaleString() : '-'}
                    </div>
                  </div>
                  <div className="fingerprint-detail-actions">
                    <button
                      className="btn-secondary"
                      onClick={() => {
                        if (runningInstance && win.url) {
                          NavigateInstanceBrowser(runningInstance.id, win.url);
                        }
                      }}
                      disabled={!win.url}
                    >
                      激活
                    </button>
                    <button
                      className="btn-danger"
                      onClick={() => closeWindow(win.id)}
                      disabled={deletingWindow === win.id}
                    >
                      {deletingWindow === win.id ? '关闭中...' : '关闭'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
            <div className="modal-actions">
              <button onClick={() => setSelectedGroup(null)}>返回</button>
            </div>
          </div>
        </div>
      )}

      {groups.length === 0 ? (
        <div className="empty-state">
          <p>暂无指纹窗口。创建一个新的指纹窗口开始使用。</p>
        </div>
      ) : (
        <div className="fingerprint-groups-list">
          {groups.map(group => (
            <div
              key={group.country}
              className="fingerprint-group-card"
              onClick={() => handleGroupClick(group)}
            >
              <div className="fingerprint-group-info">
                <span className="fingerprint-group-flag">{group.flag}</span>
                <span className="fingerprint-group-name">
                  {group.countryName} ({group.windows.length} windows)
                </span>
              </div>
              <div className="fingerprint-group-actions">
                <button
                  className="btn-danger"
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteGroup(group.country);
                  }}
                  disabled={deletingGroup === group.country}
                >
                  {deletingGroup === group.country ? '删除中...' : '删除'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
