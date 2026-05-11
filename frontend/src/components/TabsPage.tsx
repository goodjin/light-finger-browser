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

// Group tabs by instance
interface GroupedTabs {
  instance: commands.BrowserInstance;
  tabs: commands.TabInfo[];
}

export function TabsPage() {
  const [groupedTabs, setGroupedTabs] = useState<GroupedTabs[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTabId, setSelectedTabId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showNavigate, setShowNavigate] = useState(false);
  const [navigateUrl, setNavigateUrl] = useState('');
  const [creatingTab, setCreatingTab] = useState(false);
  const [selectedInstanceId, setSelectedInstanceId] = useState<string>('');
  const [newTabUrl, setNewTabUrl] = useState('');
  const [runningInstances, setRunningInstances] = useState<commands.BrowserInstance[]>([]);

  // Load all running instances and their tabs
  async function loadAllTabs() {
    try {
      setLoading(true);
      setError(null);

      // Get all instances
      const allInstances = await ListInstances(instance.InstanceFilter.createFrom({}));

      // Filter running instances only
      const running = (allInstances || []).filter(
        (inst: commands.BrowserInstance) => inst.status === 'running'
      );

      setRunningInstances(running);

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
          // Instance might have stopped, skip it
          console.warn(`Failed to load tabs for instance ${inst.id}:`, err);
        }
      }

      setGroupedTabs(results);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  // Initial load and auto-refresh every 5 seconds
  useEffect(() => {
    loadAllTabs();
    const interval = setInterval(loadAllTabs, 5000);
    return () => clearInterval(interval);
  }, []);

  // Handle tab creation
  async function createNewTab() {
    if (!selectedInstanceId) return;
    try {
      setCreatingTab(true);
      setError(null);

      // Generate a random fingerprint for the new tab
      const fp = await GenerateRandomFingerprint('US');

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
    } catch (err) {
      setError(String(err));
    } finally {
      setCreatingTab(false);
    }
  }

  // Handle tab closure
  async function handleCloseTab(instanceId: string, tabId: string, e: React.MouseEvent) {
    e.stopPropagation();
    try {
      setError(null);
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
      setError(String(err));
    }
  }

  // Handle navigation
  async function handleNavigate(instanceId: string, tabId: string) {
    if (!navigateUrl) return;
    try {
      setError(null);
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
      setError(String(err));
    }
  }

  // Open fingerprint test page
  async function handleTestFingerprint(instanceId: string, tabId: string) {
    try {
      setError(null);
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
      setError(String(err));
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
          <button className="btn-secondary" onClick={loadAllTabs} disabled={loading}>
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

      {error && (
        <div className="error-banner" style={{ color: 'red', padding: '8px', marginBottom: '8px' }}>
          {error}
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
                Leave empty to open blank tab. A random fingerprint will be assigned automatically.
              </div>
            </div>
            <div className="modal-actions">
              <button onClick={() => {
                setShowCreate(false);
                setSelectedInstanceId('');
                setNewTabUrl('');
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
