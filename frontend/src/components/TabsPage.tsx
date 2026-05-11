import { useState, useEffect } from 'react';
import {
  CreateTab,
  CloseTab,
  ListTabs,
  NavigateTab,
  GenerateRandomFingerprint,
  ListInstances,
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

// Group tabs by instance
interface GroupedTabs {
  instance: commands.BrowserInstance;
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
  const [groupedTabs, setGroupedTabs] = useState<GroupedTabs[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [errorInfo, setErrorInfo] = useState<ErrorInfo | null>(null);
  const [selectedTabId, setSelectedTabId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showNavigate, setShowNavigate] = useState(false);
  const [navigateUrl, setNavigateUrl] = useState('');
  const [creatingTab, setCreatingTab] = useState(false);
  const [selectedInstanceId, setSelectedInstanceId] = useState<string>('');
  const [newTabUrl, setNewTabUrl] = useState('');
  const [newTabCountry, setNewTabCountry] = useState<string>('US');
  const [showAdvancedConfig, setShowAdvancedConfig] = useState(false);
  const [runningInstances, setRunningInstances] = useState<commands.BrowserInstance[]>([]);
  const [cleanedTabsCount, setCleanedTabsCount] = useState(0);

  // Load all running instances and their tabs
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

      // Get tabs from each running instance
      const results: GroupedTabs[] = [];
      for (const inst of running) {
        try {
          const tabs = await ListTabs(inst.id);
          results.push({
            instance: inst,
            tabs: tabs || [],
          });
        } catch (err) {
          // Instance might have stopped during refresh, skip it (silently)
          console.warn(`Failed to load tabs for instance ${inst.id}:`, err);
        }
      }

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

  // Initial load and auto-refresh every 5 seconds (background refresh)
  useEffect(() => {
    loadAllTabs(false); // Initial load (shows loading state, may show errors)
    const interval = setInterval(() => loadAllTabs(true), 5000); // Background refresh (silent)
    return () => clearInterval(interval);
  }, []);

  // Handle tab creation
  async function createNewTab() {
    if (!selectedInstanceId) return;
    try {
      setCreatingTab(true);
      setError(null);
      setErrorInfo(null);

      // Generate a random fingerprint with the selected country
      const fp = await GenerateRandomFingerprint(newTabCountry);

      // Create the tab (result stored for potential future use)
      await CreateTab(selectedInstanceId, commands.TabConfig.createFrom({
        URL: newTabUrl || 'about:blank',
        Fingerprint: fp,
        ProxyURL: '',
      }));

      // Refresh to show new tab
      await loadAllTabs();

      // Close dialogs
      setShowCreate(false);
      setSelectedInstanceId('');
      setNewTabUrl('');
      // Keep the country selection for convenience
    } catch (err) {
      const errorType = classifyError(err);
      const displayMessage = getErrorDisplayMessage(err, errorType);
      
      console.error('[TabsPage] Failed to create tab:', {
        instanceId: selectedInstanceId,
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

      // Update local state
      setGroupedTabs(prev =>
        prev.map(group => {
          if (group.instance.id === instanceId) {
            return {
              ...group,
              tabs: group.tabs.filter(t => t.ID !== tabId),
            };
          }
          return group;
        }).filter(group => group.tabs.length > 0)
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

      // Update local state
      setGroupedTabs(prev =>
        prev.map(group => {
          if (group.instance.id === instanceId) {
            return {
              ...group,
              tabs: group.tabs.map(t =>
                t.ID === tabId ? commands.TabInfo.createFrom({ ...t, URL: navigateUrl }) : t
              ),
            };
          }
          return group;
        })
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

      // Update local state
      setGroupedTabs(prev =>
        prev.map(group => {
          if (group.instance.id === instanceId) {
            return {
              ...group,
              tabs: group.tabs.map(t =>
                t.ID === tabId ? commands.TabInfo.createFrom({ ...t, URL: FINGERPRINT_SERVER_URL }) : t
              ),
            };
          }
          return group;
        })
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
      <div className="page-header">
        <div className="header-left">
          <h2>Browser Tabs</h2>
          <span className="tab-count">
            {totalTabs} tab{totalTabs !== 1 ? 's' : ''} across {groupedTabs.length} instance{groupedTabs.length !== 1 ? 's' : ''}
          </span>
        </div>
        <div className="header-right">
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
              <label>Instance</label>
              <select
                value={selectedInstanceId}
                onChange={e => setSelectedInstanceId(e.target.value)}
                disabled={creatingTab}
              >
                <option value="">Select an instance...</option>
                {runningInstances.map(inst => (
                  <option key={inst.id} value={inst.id}>
                    {inst.name || `${inst.id.slice(0, 8)}...`} ({inst.status})
                  </option>
                ))}
              </select>
              {runningInstances.length === 0 && (
                <div className="helper-text" style={{ color: '#f97316' }}>
                  No running instances available. Start an instance first.
                </div>
              )}
            </div>
            <div className="form-group">
              <label>URL (optional)</label>
              <input
                type="text"
                placeholder="https://example.com"
                value={newTabUrl}
                onChange={e => setNewTabUrl(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && selectedInstanceId && createNewTab()}
                disabled={creatingTab}
              />
              <div className="helper-text">
                Leave empty to open blank tab.
              </div>
            </div>
            
            {/* Fingerprint Configuration */}
            <div className="form-group">
              <label style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span>Fingerprint Country</span>
                <span style={{ 
                  fontSize: '11px', 
                  color: '#1976d2',
                  background: '#e3f2fd',
                  padding: '2px 8px',
                  borderRadius: '4px',
                  fontWeight: 'normal',
                }}>
                  Required
                </span>
              </label>
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
                setSelectedInstanceId('');
                setNewTabUrl('');
                setShowAdvancedConfig(false);
              }} disabled={creatingTab}>
                Cancel
              </button>
              <button
                className="btn-primary"
                onClick={createNewTab}
                disabled={!selectedInstanceId || creatingTab}
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
                    const group = groupedTabs.find(g => g.tabs.some(t => t.ID === selectedTabId));
                    if (group) {
                      handleNavigate(group.instance.id, selectedTabId);
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
                  const group = groupedTabs.find(g => g.tabs.some(t => t.ID === selectedTabId));
                  if (group) {
                    handleNavigate(group.instance.id, selectedTabId);
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

      {/* Tabs List Grouped by Instance */}
      <div className="tabs-groups">
        {groupedTabs.map(group => (
          <div key={group.instance.id} className="instance-group" style={{
            marginBottom: '24px',
            border: '1px solid #e5e7eb',
            borderRadius: '8px',
            overflow: 'hidden',
          }}>
            {/* Instance Header */}
            <div className="group-header" style={{
              background: '#f9fafb',
              padding: '12px 16px',
              borderBottom: '1px solid #e5e7eb',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span style={{
                  fontWeight: '600',
                  color: '#1976d2',
                }}>
                  {group.instance.name || `${group.instance.id.slice(0, 8)}...`}
                </span>
                <span style={{
                  background: '#22c55e',
                  color: 'white',
                  padding: '2px 8px',
                  borderRadius: '4px',
                  fontSize: '11px',
                }}>
                  {group.instance.status}
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
                    setSelectedInstanceId(group.instance.id);
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
                  No tabs in this instance
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
                        onClick={() => handleTestFingerprint(group.instance.id, tab.ID)}
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
                        onClick={(e) => handleCloseTab(group.instance.id, tab.ID, e)}
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
