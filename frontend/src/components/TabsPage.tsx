import { useState, useEffect } from 'react';
import {
  CreateTab,
  CloseTab,
  ListTabs,
  NavigateTab,
  GenerateRandomFingerprint,
  ListInstances,
  GetAccessLogs,
} from '../wailsjs/go/main/App';
import { commands, instance } from '../wailsjs/go/models';

const FINGERPRINT_SERVER_URL = 'http://localhost:18080/';

// Error type classification for better user feedback
type ErrorType = 'network' | 'timeout' | 'not_found' | 'unknown';

interface ErrorInfo {
  message: string;
  type: ErrorType;
  timestamp: Date;
}

// Group tabs by fingerprint country
interface FingerprintCountryGroup {
  country: string;  // Country code (e.g., "US", "GB", "DE")
  countryName: string;
  flag: string;
  tabs: commands.TabInfo[];
}

// Classify error type from error message
function classifyError(error: unknown): ErrorType {
  const msg = String(error).toLowerCase();
  if (msg.includes('timeout') || msg.includes('timed out') || msg.includes('8s')) {
    return 'timeout';
  }
  if (msg.includes('network') || msg.includes('connection') || msg.includes('refused') || msg.includes('econnreset')) {
    return 'network';
  }
  if (msg.includes('not found') || msg.includes('404') || msg.includes('enoent')) {
    return 'not_found';
  }
  return 'unknown';
}

// Get user-friendly error message based on error type
function getErrorDisplayMessage(error: unknown, type: ErrorType): string {
  const originalMsg = String(error);
  
  switch (type) {
    case 'network':
      return `网络连接失败: ${originalMsg}`;
    case 'timeout':
      return `请求超时: ${originalMsg}`;
    case 'not_found':
      return `资源未找到: ${originalMsg}`;
    default:
      return originalMsg;
  }
}

// Supported countries for fingerprint configuration
const SUPPORTED_COUNTRIES = [
  { code: 'US', name: 'United States', flag: '🇺🇸', timezone: 'America/New_York' },
  { code: 'GB', name: 'United Kingdom', flag: '🇬🇧', timezone: 'Europe/London' },
  { code: 'DE', name: 'Germany', flag: '🇩🇪', timezone: 'Europe/Berlin' },
  { code: 'FR', name: 'France', flag: '🇫🇷', timezone: 'Europe/Paris' },
  { code: 'JP', name: 'Japan', flag: '🇯🇵', timezone: 'Asia/Tokyo' },
  { code: 'CN', name: 'China', flag: '🇨🇳', timezone: 'Asia/Shanghai' },
  { code: 'CA', name: 'Canada', flag: '🇨🇦', timezone: 'America/Toronto' },
  { code: 'AU', name: 'Australia', flag: '🇦🇺', timezone: 'Australia/Sydney' },
  { code: 'BR', name: 'Brazil', flag: '🇧🇷', timezone: 'America/Sao_Paulo' },
  { code: 'IN', name: 'India', flag: '🇮🇳', timezone: 'Asia/Kolkata' },
  { code: 'IT', name: 'Italy', flag: '🇮🇹', timezone: 'Europe/Rome' },
  { code: 'ES', name: 'Spain', flag: '🇪🇸', timezone: 'Europe/Madrid' },
];

export function TabsPage() {
  const [groupedTabs, setGroupedTabs] = useState<FingerprintCountryGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [errorInfo, setErrorInfo] = useState<ErrorInfo | null>(null);
  const [selectedTabId, setSelectedTabId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showNavigate, setShowNavigate] = useState(false);
  const [showAccessLogs, setShowAccessLogs] = useState(false);
  const [navigateUrl, setNavigateUrl] = useState('');
  const [creatingTab, setCreatingTab] = useState(false);
  const [newTabUrl, setNewTabUrl] = useState('');
  const [newTabCountry, setNewTabCountry] = useState<string>('US');
  const [showAdvancedConfig, setShowAdvancedConfig] = useState(false);
  const [runningInstances, setRunningInstances] = useState<commands.BrowserInstance[]>([]);
  const [cleanedTabsCount, setCleanedTabsCount] = useState(0);
  const [accessLogs, setAccessLogs] = useState<commands.AccessLogInfo[]>([]);
  const [accessLogsLoading, setAccessLogsLoading] = useState(false);
  const [accessLogsFilter, setAccessLogsFilter] = useState<string>('');

  // Helper function to get country info from country code
  function getCountryInfo(code: string): { name: string; flag: string } {
    const country = SUPPORTED_COUNTRIES.find(c => c.code === code);
    return country ? { name: country.name, flag: country.flag } : { name: code, flag: '🏳️' };
  }

  // Load all running instances and their tabs, grouped by fingerprint country
  // isBackgroundRefresh: if true, errors are silently ignored and user selection is preserved
  async function loadAllTabs(isBackgroundRefresh: boolean = false) {
    try {
      if (!isBackgroundRefresh) {
        setLoading(true);
        setError(null);
        setErrorInfo(null);
      }

      // Get all instances
      const allInstances = await ListInstances(instance.InstanceFilter.createFrom({}));

      // Filter running instances only
      const running = (allInstances || []).filter(
        (inst: commands.BrowserInstance) => inst.status === 'running'
      );

      if (!isBackgroundRefresh) {
        setRunningInstances(running);
      }

      // Track how many tabs were cleaned up (instances that stopped)
      let tabsCleanedCount = 0;
      const previousTabIds = new Set<string>();
      groupedTabs.forEach(group => {
        group.tabs.forEach(tab => previousTabIds.add(tab.ID));
      });

      // Get tabs from each running instance and group by fingerprint country
      const countryMap = new Map<string, commands.TabInfo[]>();
      
      for (const inst of running) {
        try {
          const tabs = await ListTabs(inst.id);
          // Group tabs by fingerprint country
          for (const tab of tabs || []) {
            const country = tab.FingerprintCountry || 'US'; // Default to US if not set
            if (!countryMap.has(country)) {
              countryMap.set(country, []);
            }
            countryMap.get(country)!.push(tab);
          }
        } catch (err) {
          // Instance might have stopped during refresh, skip it (silently)
          console.warn(`Failed to load tabs for instance ${inst.id}:`, err);
        }
      }

      // Convert map to FingerprintCountryGroup array
      const results: FingerprintCountryGroup[] = [];
      for (const [country, tabs] of countryMap) {
        const countryInfo = getCountryInfo(country);
        results.push({
          country,
          countryName: countryInfo.name,
          flag: countryInfo.flag,
          tabs,
        });
      }

      // Sort groups by country name
      results.sort((a, b) => a.countryName.localeCompare(b.countryName));

      // Count tabs that were cleaned up (instances that stopped or tabs that no longer exist)
      results.forEach(group => {
        group.tabs.forEach(tab => previousTabIds.delete(tab.ID));
      });
      tabsCleanedCount = previousTabIds.size;
      if (tabsCleanedCount > 0) {
        setCleanedTabsCount(prev => prev + tabsCleanedCount);
      }

      // Preserve user selection state during background refresh
      if (isBackgroundRefresh && selectedTabId) {
        // Check if selected tab still exists
        const tabStillExists = results.some(group =>
          group.tabs.some(tab => tab.ID === selectedTabId)
        );
        if (!tabStillExists) {
          setSelectedTabId(null);
        }
      }

      setGroupedTabs(results);
    } catch (err) {
      // Classify error and provide user-friendly message
      const errorType = classifyError(err);
      const displayMessage = getErrorDisplayMessage(err, errorType);
      
      // Log error for debugging
      console.error('[TabsPage] Failed to load tabs:', {
        error: err,
        type: errorType,
        timestamp: new Date().toISOString(),
        isBackgroundRefresh,
      });
      
      // Silent error handling during background refresh - don't show errors to user
      if (!isBackgroundRefresh) {
        setError(displayMessage);
        setErrorInfo({
          message: displayMessage,
          type: errorType,
          timestamp: new Date(),
        });
      }
    } finally {
      if (!isBackgroundRefresh) {
        setLoading(false);
      }
    }
  }

  // Load access logs
  async function loadAccessLogs(tabID?: string) {
    try {
      setAccessLogsLoading(true);
      const query = commands.AccessLogQuery.createFrom({
        TabID: tabID || '',
        StartTime: '',
        EndTime: '',
      });
      const logs = await GetAccessLogs(query);
      setAccessLogs(logs || []);
    } catch (err) {
      console.error('[TabsPage] Failed to load access logs:', err);
    } finally {
      setAccessLogsLoading(false);
    }
  }

  // Initial load and auto-refresh every 5 seconds (background refresh)
  useEffect(() => {
    loadAllTabs(false); // Initial load (shows loading state, may show errors)
    const interval = setInterval(() => loadAllTabs(true), 5000); // Background refresh (silent)
    return () => clearInterval(interval);
  }, []);

  // Handle tab creation
  async function createNewTab() {
    // In single instance mode, use the first running instance
    if (runningInstances.length === 0) {
      setError('No running instances. Please start an instance first.');
      setErrorInfo({
        message: 'No running instances available.',
        type: 'not_found',
        timestamp: new Date(),
      });
      return;
    }

    const instanceId = runningInstances[0].id;
    try {
      setCreatingTab(true);
      setError(null);
      setErrorInfo(null);

      // Generate a random fingerprint with the selected country
      const fp = await GenerateRandomFingerprint(newTabCountry);

      // Create the tab (result stored for potential future use)
      await CreateTab(instanceId, commands.TabConfig.createFrom({
        URL: newTabUrl || 'about:blank',
        Fingerprint: fp,
        ProxyURL: '',
        Country: newTabCountry,
      }));

      // Refresh to show new tab
      await loadAllTabs();

      // Close dialogs
      setShowCreate(false);
      setNewTabUrl('');
      // Keep the country selection for convenience
    } catch (err) {
      const errorType = classifyError(err);
      const displayMessage = getErrorDisplayMessage(err, errorType);
      
      console.error('[TabsPage] Failed to create tab:', {
        instanceId,
        url: newTabUrl,
        error: err,
        type: errorType,
        timestamp: new Date().toISOString(),
      });
      
      setError(displayMessage);
      setErrorInfo({
        message: displayMessage,
        type: errorType,
        timestamp: new Date(),
      });
    } finally {
      setCreatingTab(false);
    }
  }

  // Handle tab closure
  async function handleCloseTab(instanceId: string, tabId: string, e: React.MouseEvent) {
    e.stopPropagation();
    try {
      setError(null);
      setErrorInfo(null);
      await CloseTab(instanceId, tabId);

      // Update local state - remove tab from its country group
      setGroupedTabs(prev =>
        prev.map(group => ({
          ...group,
          tabs: group.tabs.filter(t => t.ID !== tabId),
        })).filter(group => group.tabs.length > 0)
      );

      if (selectedTabId === tabId) {
        setSelectedTabId(null);
      }
    } catch (err) {
      const errorType = classifyError(err);
      const displayMessage = getErrorDisplayMessage(err, errorType);
      
      console.error('[TabsPage] Failed to close tab:', {
        instanceId,
        tabId,
        error: err,
        type: errorType,
        timestamp: new Date().toISOString(),
      });
      
      setError(displayMessage);
      setErrorInfo({
        message: displayMessage,
        type: errorType,
        timestamp: new Date(),
      });
    }
  }

  // Handle navigation
  async function handleNavigate(instanceId: string, tabId: string) {
    if (!navigateUrl) return;
    try {
      setError(null);
      setErrorInfo(null);
      await NavigateTab(instanceId, tabId, navigateUrl);

      // Update local state - update tab URL in its country group
      setGroupedTabs(prev =>
        prev.map(group => ({
          ...group,
          tabs: group.tabs.map(t =>
            t.ID === tabId ? commands.TabInfo.createFrom({ ...t, URL: navigateUrl }) : t
          ),
        }))
      );

      setShowNavigate(false);
      setNavigateUrl('');
    } catch (err) {
      const errorType = classifyError(err);
      const displayMessage = getErrorDisplayMessage(err, errorType);
      
      console.error('[TabsPage] Failed to navigate tab:', {
        instanceId,
        tabId,
        url: navigateUrl,
        error: err,
        type: errorType,
        timestamp: new Date().toISOString(),
      });
      
      setError(displayMessage);
      setErrorInfo({
        message: displayMessage,
        type: errorType,
        timestamp: new Date(),
      });
    }
  }

  // Open fingerprint test page
  async function handleTestFingerprint(instanceId: string, tabId: string) {
    try {
      setError(null);
      setErrorInfo(null);
      await NavigateTab(instanceId, tabId, FINGERPRINT_SERVER_URL);

      // Update local state - update tab URL in its country group
      setGroupedTabs(prev =>
        prev.map(group => ({
          ...group,
          tabs: group.tabs.map(t =>
            t.ID === tabId ? commands.TabInfo.createFrom({ ...t, URL: FINGERPRINT_SERVER_URL }) : t
          ),
        }))
      );
    } catch (err) {
      const errorType = classifyError(err);
      const displayMessage = getErrorDisplayMessage(err, errorType);
      
      console.error('[TabsPage] Failed to test fingerprint:', {
        instanceId,
        tabId,
        error: err,
        type: errorType,
        timestamp: new Date().toISOString(),
      });
      
      setError(displayMessage);
      setErrorInfo({
        message: displayMessage,
        type: errorType,
        timestamp: new Date(),
      });
    }
  }

  // Get total tab count
  const totalTabs = groupedTabs.reduce((sum, group) => sum + group.tabs.length, 0);

  // Check if empty state
  const isEmpty = !loading && groupedTabs.length === 0;

  // Format date for display
  function formatDate(date: any): string {
    if (!date) return '-';
    try {
      const d = new Date(date);
      return d.toLocaleString();
    } catch {
      return '-';
    }
  }

  // Truncate URL for display
  function truncateUrl(url: string | undefined, maxLength: number = 50): string {
    if (!url) return 'about:blank';
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + '...';
  }

  if (loading && groupedTabs.length === 0) {
    return (
      <div className="tabs-page">
        <div className="loading">Loading tabs...</div>
      </div>
    );
  }

  return (
    <div className="tabs-page">
      {/* Instance Status Section */}
      {runningInstances.length > 0 && (
        <div className="instance-status-bar" style={{
          display: 'flex',
          alignItems: 'center',
          gap: '16px',
          padding: '12px 16px',
          marginBottom: '16px',
          background: runningInstances[0].status === 'running' ? '#f0fdf4' : '#fef2f2',
          border: `1px solid ${runningInstances[0].status === 'running' ? '#86efac' : '#fecaca'}`,
          borderRadius: '8px',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{
              width: '10px',
              height: '10px',
              borderRadius: '50%',
              background: runningInstances[0].status === 'running' ? '#22c55e' : '#ef4444',
            }} />
            <span style={{ fontWeight: '500', color: runningInstances[0].status === 'running' ? '#166534' : '#991b1b' }}>
              {runningInstances[0].status === 'running' ? 'Running' : 'Stopped'}
            </span>
          </div>
          <div style={{ color: '#666', fontSize: '13px' }}>
            <span style={{ marginRight: '8px' }}>Instance:</span>
            <span style={{ fontFamily: 'monospace' }}>
              {runningInstances[0].name || `${runningInstances[0].id.slice(0, 8)}...`}
            </span>
          </div>
          {runningInstances[0].pid > 0 && (
            <div style={{ color: '#666', fontSize: '13px' }}>
              <span style={{ marginRight: '8px' }}>PID:</span>
              <span style={{ fontFamily: 'monospace' }}>{runningInstances[0].pid}</span>
            </div>
          )}
          {runningInstances[0].port > 0 && (
            <div style={{ color: '#666', fontSize: '13px' }}>
              <span style={{ marginRight: '8px' }}>CDP Port:</span>
              <span style={{ fontFamily: 'monospace' }}>{runningInstances[0].port}</span>
            </div>
          )}
          <div style={{ color: '#666', fontSize: '13px' }}>
            <span style={{ marginRight: '8px' }}>Tabs:</span>
            <span style={{ fontWeight: '500' }}>{totalTabs}</span>
          </div>
        </div>
      )}

      <div className="page-header">
        <div className="header-left">
          <h2>Browser Tabs</h2>
          <span className="tab-count">
            {totalTabs} tab{totalTabs !== 1 ? 's' : ''} across {groupedTabs.length} fingerprint group{groupedTabs.length !== 1 ? 's' : ''}
          </span>
        </div>
        <div className="header-right">
          <button 
            className="btn-secondary" 
            onClick={() => {
              setShowAccessLogs(true);
              loadAccessLogs();
            }}
            style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
          >
            📋 Access Logs
          </button>
          <button className="btn-secondary" onClick={() => loadAllTabs(false)} disabled={loading}>
            Refresh
          </button>
          <button
            className="btn-primary"
            onClick={() => setShowCreate(true)}
            disabled={runningInstances.length === 0}
          >
            + New Tab
          </button>
        </div>
      </div>

      {error && errorInfo && (
        <div 
          className="error-banner" 
          style={{ 
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '12px 16px', 
            marginBottom: '8px',
            background: '#fef2f2',
            border: '1px solid #fecaca',
            borderRadius: '6px',
            color: '#dc2626',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ fontSize: '18px' }}>⚠️</span>
            <div>
              <div style={{ fontWeight: '500' }}>操作失败</div>
              <div style={{ fontSize: '13px', color: '#991b1b', marginTop: '2px' }}>{error}</div>
            </div>
          </div>
          <button 
            className="btn-secondary"
            onClick={() => {
              setError(null);
              setErrorInfo(null);
              loadAllTabs(false);
            }}
            style={{ 
              fontSize: '12px', 
              padding: '6px 12px',
              background: '#fff',
              border: '1px solid #fecaca',
            }}
          >
            重试
          </button>
        </div>
      )}

      {/* Auto-cleanup notification */}
      {cleanedTabsCount > 0 && (
        <div 
          style={{ 
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '10px 16px',
            marginBottom: '8px',
            background: '#fff7ed',
            border: '1px solid #fed7aa',
            borderRadius: '6px',
            color: '#c2410c',
            fontSize: '13px',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span>🔄</span>
            <span>已自动清理 <strong>{cleanedTabsCount}</strong> 个标签页（实例已停止）</span>
          </div>
          <button 
            onClick={() => setCleanedTabsCount(0)}
            style={{ 
              background: 'transparent', 
              border: 'none', 
              cursor: 'pointer',
              color: '#c2410c',
              fontSize: '16px',
            }}
          >
            ×
          </button>
        </div>
      )}

      {/* Create Tab Modal */}
      {showCreate && (
        <div className="modal-overlay" onClick={() => setShowCreate(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h3>Create New Tab</h3>
            <div className="form-group">
              <label>Fingerprint Country</label>
              <div className="country-selector">
                <select
                  value={newTabCountry}
                  onChange={e => setNewTabCountry(e.target.value)}
                  disabled={creatingTab}
                  style={{
                    width: '100%',
                    padding: '10px 12px',
                    border: '1px solid #e5e7eb',
                    borderRadius: '6px',
                    fontSize: '14px',
                    background: '#fff',
                    cursor: 'pointer',
                  }}
                >
                  {SUPPORTED_COUNTRIES.map(country => (
                    <option key={country.code} value={country.code}>
                      {country.flag} {country.name} ({country.code})
                    </option>
                  ))}
                </select>
              </div>
              <div className="helper-text">
                Each tab will have a unique fingerprint based on the selected country.
              </div>
            </div>

            <div className="form-group">
              <label>URL (optional)</label>
              <input
                type="text"
                placeholder="https://example.com"
                value={newTabUrl}
                onChange={e => setNewTabUrl(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && createNewTab()}
                disabled={creatingTab}
              />
              <div className="helper-text">
                Leave empty to open blank tab.
              </div>
            </div>

            {/* Advanced Configuration Toggle */}
            <div className="form-group" style={{ marginTop: '8px' }}>
              <button
                type="button"
                onClick={() => setShowAdvancedConfig(!showAdvancedConfig)}
                style={{
                  background: 'transparent',
                  border: 'none',
                  color: '#666',
                  fontSize: '13px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  padding: '4px 0',
                }}
              >
                <span style={{ 
                  transform: showAdvancedConfig ? 'rotate(90deg)' : 'rotate(0deg)',
                  transition: 'transform 0.2s',
                  display: 'inline-block',
                }}>
                  ▶
                </span>
                Advanced Configuration
              </button>
              
              {showAdvancedConfig && (
                <div style={{
                  marginTop: '12px',
                  padding: '12px',
                  background: '#f9fafb',
                  borderRadius: '6px',
                  border: '1px solid #e5e7eb',
                }}>
                  <div style={{ fontSize: '12px', color: '#666', marginBottom: '8px' }}>
                    Selected fingerprint settings for <strong>{SUPPORTED_COUNTRIES.find(c => c.code === newTabCountry)?.name}</strong>:
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', fontSize: '12px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span style={{ color: '#888' }}>Timezone:</span>
                      <span style={{ fontFamily: 'monospace' }}>{SUPPORTED_COUNTRIES.find(c => c.code === newTabCountry)?.timezone}</span>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span style={{ color: '#888' }}>Platform:</span>
                      <span style={{ fontFamily: 'monospace' }}>Windows</span>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span style={{ color: '#888' }}>Screen:</span>
                      <span style={{ fontFamily: 'monospace' }}>Random (1920x1080, 1366x768, etc.)</span>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span style={{ color: '#888' }}>Locale:</span>
                      <span style={{ fontFamily: 'monospace' }}>{newTabCountry === 'US' ? 'en-US' : SUPPORTED_COUNTRIES.find(c => c.code === newTabCountry)?.code.toLowerCase() + '-' + SUPPORTED_COUNTRIES.find(c => c.code === newTabCountry)?.code.toUpperCase()}</span>
                    </div>
                  </div>
                </div>
              )}
            </div>

            <div className="modal-actions">
              <button onClick={() => {
                setShowCreate(false);
                setNewTabUrl('');
                setShowAdvancedConfig(false);
              }} disabled={creatingTab}>
                Cancel
              </button>
              <button
                className="btn-primary"
                onClick={createNewTab}
                disabled={creatingTab}
              >
                {creatingTab ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Navigate Modal */}
      {showNavigate && selectedTabId && (
        <div className="modal-overlay" onClick={() => setShowNavigate(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h3>Navigate Tab</h3>
            <div className="form-group">
              <label>URL</label>
              <input
                type="text"
                placeholder="https://example.com"
                value={navigateUrl}
                onChange={e => setNavigateUrl(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter' && navigateUrl) {
                    const tab = groupedTabs.flatMap(g => g.tabs).find(t => t.ID === selectedTabId);
                    if (tab) {
                      handleNavigate(tab.InstanceID, selectedTabId);
                    }
                  }
                }}
              />
            </div>
            <div className="modal-actions">
              <button onClick={() => {
                setShowNavigate(false);
                setNavigateUrl('');
              }}>
                Cancel
              </button>
              <button
                className="btn-primary"
                onClick={() => {
                  const tab = groupedTabs.flatMap(g => g.tabs).find(t => t.ID === selectedTabId);
                  if (tab) {
                    handleNavigate(tab.InstanceID, selectedTabId);
                  }
                }}
                disabled={!navigateUrl}
              >
                Navigate
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Access Log Panel */}
      {showAccessLogs && (
        <div className="modal-overlay" onClick={() => setShowAccessLogs(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: '800px', width: '90%', maxHeight: '80vh', display: 'flex', flexDirection: 'column' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <h3>📋 Access Logs</h3>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <input
                  type="text"
                  placeholder="Filter by tab ID..."
                  value={accessLogsFilter}
                  onChange={e => setAccessLogsFilter(e.target.value)}
                  style={{
                    padding: '6px 12px',
                    border: '1px solid #e5e7eb',
                    borderRadius: '4px',
                    fontSize: '13px',
                    width: '200px',
                  }}
                />
                <button 
                  className="btn-secondary" 
                  onClick={() => loadAccessLogs()}
                  style={{ fontSize: '12px' }}
                >
                  Refresh
                </button>
                <button 
                  onClick={() => setShowAccessLogs(false)}
                  style={{
                    background: 'transparent',
                    border: 'none',
                    fontSize: '20px',
                    cursor: 'pointer',
                    color: '#666',
                  }}
                >
                  ×
                </button>
              </div>
            </div>
            
            {accessLogsLoading ? (
              <div style={{ textAlign: 'center', padding: '32px', color: '#666' }}>
                Loading access logs...
              </div>
            ) : accessLogs.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '32px', color: '#666' }}>
                <div style={{ fontSize: '48px', marginBottom: '16px' }}>📋</div>
                <p>No access logs found.</p>
                <p style={{ fontSize: '13px', marginTop: '8px' }}>
                  Access logs are recorded when you navigate to a URL in a tab.
                </p>
              </div>
            ) : (
              <div style={{ flex: 1, overflowY: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                  <thead>
                    <tr style={{ background: '#f9fafb', borderBottom: '2px solid #e5e7eb' }}>
                      <th style={{ padding: '10px 12px', textAlign: 'left', fontWeight: '600' }}>Time</th>
                      <th style={{ padding: '10px 12px', textAlign: 'left', fontWeight: '600' }}>Tab ID</th>
                      <th style={{ padding: '10px 12px', textAlign: 'left', fontWeight: '600' }}>URL</th>
                      <th style={{ padding: '10px 12px', textAlign: 'left', fontWeight: '600' }}>Title</th>
                      <th style={{ padding: '10px 12px', textAlign: 'right', fontWeight: '600' }}>Duration</th>
                    </tr>
                  </thead>
                  <tbody>
                    {accessLogs
                      .filter(log => !accessLogsFilter || log.TabID.includes(accessLogsFilter))
                      .map(log => (
                        <tr key={log.ID} style={{ borderBottom: '1px solid #f3f4f6' }}>
                          <td style={{ padding: '10px 12px', color: '#666', fontFamily: 'monospace', fontSize: '12px' }}>
                            {formatDate(log.VisitedAt)}
                          </td>
                          <td style={{ padding: '10px 12px', fontFamily: 'monospace', fontSize: '12px' }}>
                            {log.TabID.slice(0, 8)}...
                          </td>
                          <td style={{ padding: '10px 12px', fontFamily: 'monospace', fontSize: '12px', maxWidth: '250px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {log.URL ? truncateUrl(log.URL, 40) : '-'}
                          </td>
                          <td style={{ padding: '10px 12px', maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {log.Title || '-'}
                          </td>
                          <td style={{ padding: '10px 12px', textAlign: 'right', color: '#666', fontSize: '12px' }}>
                            {log.DurationMs > 0 ? `${(log.DurationMs / 1000).toFixed(1)}s` : '-'}
                          </td>
                        </tr>
                      ))}
                  </tbody>
                </table>
              </div>
            )}
            <div style={{ marginTop: '16px', paddingTop: '12px', borderTop: '1px solid #e5e7eb', color: '#666', fontSize: '12px' }}>
              Total: {accessLogs.length} log entries
            </div>
          </div>
        </div>
      )}

      {/* Empty State */}
      {isEmpty && (
        <div className="empty-state" style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '48px',
          color: '#666',
        }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>📑</div>
          <h3 style={{ margin: '0 0 8px 0' }}>暂无标签页</h3>
          <p style={{ margin: '0 0 16px 0' }}>启动实例后可管理标签页</p>
          <button
            className="btn-primary"
            onClick={() => {
              // Navigate to instances page would require a callback
              // For now, just show the create dialog if there are running instances
              if (runningInstances.length > 0) {
                setShowCreate(true);
              }
            }}
            disabled={runningInstances.length === 0}
          >
            + New Tab
          </button>
        </div>
      )}

      {/* Tabs List Grouped by Fingerprint Country */}
      <div className="tabs-groups">
        {groupedTabs.map(group => (
          <div key={group.country} className="fingerprint-country-group" style={{
            marginBottom: '24px',
            border: '1px solid #e5e7eb',
            borderRadius: '8px',
            overflow: 'hidden',
          }}>
            {/* Fingerprint Country Header */}
            <div className="group-header" style={{
              background: '#f9fafb',
              padding: '12px 16px',
              borderBottom: '1px solid #e5e7eb',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span style={{ fontSize: '24px' }}>{group.flag}</span>
                <span style={{
                  fontWeight: '600',
                  color: '#1976d2',
                }}>
                  {group.countryName} ({group.country})
                </span>
                <span style={{ color: '#888', fontSize: '12px' }}>
                  {group.tabs.length} tab{group.tabs.length !== 1 ? 's' : ''}
                </span>
              </div>
              <div style={{ display: 'flex', gap: '8px' }}>
                <button
                  className="btn-secondary"
                  style={{ fontSize: '11px', padding: '4px 8px' }}
                  onClick={() => {
                    setNewTabCountry(group.country);
                    setShowCreate(true);
                  }}
                >
                  + New Tab
                </button>
              </div>
            </div>

            {/* Tabs List */}
            <div className="tabs-list" style={{ padding: '8px' }}>
              {group.tabs.length === 0 ? (
                <div style={{
                  padding: '24px',
                  textAlign: 'center',
                  color: '#888',
                }}>
                  No tabs in this country group
                </div>
              ) : (
                group.tabs.map(tab => (
                  <div
                    key={tab.ID}
                    className={`tab-card ${selectedTabId === tab.ID ? 'selected' : ''}`}
                    style={{
                      display: 'flex',
                      alignItems: 'flex-start',
                      gap: '12px',
                      padding: '12px',
                      marginBottom: '8px',
                      border: selectedTabId === tab.ID ? '2px solid #1976d2' : '1px solid #e5e7eb',
                      borderRadius: '6px',
                      cursor: 'pointer',
                      background: selectedTabId === tab.ID ? '#e3f2fd' : '#fff',
                      transition: 'all 0.15s ease',
                    }}
                    onClick={() => setSelectedTabId(tab.ID)}
                  >
                    {/* Fingerprint Seed */}
                    <div style={{
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      minWidth: '80px',
                    }}>
                      <span style={{
                        fontFamily: 'monospace',
                        fontSize: '11px',
                        color: '#666',
                        background: '#f3f4f6',
                        padding: '2px 6px',
                        borderRadius: '3px',
                      }}>
                        {tab.FingerprintSeed?.slice(0, 8) || 'N/A'}
                      </span>
                      <span style={{ fontSize: '10px', color: '#888', marginTop: '2px' }}>
                        FP Seed
                      </span>
                    </div>

                    {/* URL and Title */}
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{
                        fontWeight: '500',
                        color: '#333',
                        marginBottom: '4px',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}>
                        {tab.Title || 'Untitled'}
                      </div>
                      <div style={{
                        fontSize: '12px',
                        color: '#666',
                        fontFamily: 'monospace',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}>
                        {truncateUrl(tab.URL)}
                      </div>
                      <div style={{
                        fontSize: '11px',
                        color: '#888',
                        marginTop: '4px',
                        display: 'flex',
                        gap: '12px',
                      }}>
                        <span>Created: {formatDate(tab.CreatedAt)}</span>
                        <span>Active: {formatDate(tab.LastActiveAt)}</span>
                      </div>
                    </div>

                    {/* Actions */}
                    <div style={{
                      display: 'flex',
                      gap: '4px',
                      flexShrink: 0,
                    }}>
                      <button
                        className="btn-secondary"
                        style={{ fontSize: '11px', padding: '4px 8px' }}
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelectedTabId(tab.ID);
                          setNavigateUrl(tab.URL || '');
                          setShowNavigate(true);
                        }}
                        title="Navigate"
                      >
                        →
                      </button>
                      <button
                        className="btn-secondary"
                        style={{ fontSize: '11px', padding: '4px 8px' }}
                        onClick={() => handleTestFingerprint(tab.InstanceID, tab.ID)}
                        title="Test Fingerprint"
                      >
                        FP
                      </button>
                      <button
                        style={{
                          fontSize: '11px',
                          padding: '4px 8px',
                          color: '#ef4444',
                          background: 'transparent',
                          border: '1px solid #ef4444',
                          borderRadius: '4px',
                          cursor: 'pointer',
                        }}
                        onClick={(e) => handleCloseTab(tab.InstanceID, tab.ID, e)}
                        title="Close"
                      >
                        ×
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
