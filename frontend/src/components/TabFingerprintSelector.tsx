import { useState, useEffect } from 'react';
import {
    CreateTab,
    CloseTab,
    ListTabs,
    NavigateTab,
    GenerateRandomFingerprint,
} from '../wailsjs/go/main/App';
import { commands } from '../wailsjs/go/models';

type TabFingerprintSelectorProps = {
    instanceId: string;
    onTabCreated?: (tab: commands.TabInfo) => void;
    onTabClosed?: (tabId: string) => void;
    onTabNavigated?: (tabId: string, url: string) => void;
};

export function TabFingerprintSelector({
    instanceId,
    onTabCreated,
    onTabClosed,
    onTabNavigated,
}: TabFingerprintSelectorProps) {
    const [tabs, setTabs] = useState<commands.TabInfo[]>([]);
    const [loading, setLoading] = useState(false);
    const [selectedTabId, setSelectedTabId] = useState<string | null>(null);
    const [showCreate, setShowCreate] = useState(false);
    const [url, setUrl] = useState('');
    const [navigateUrl, setNavigateUrl] = useState('');
    const [showNavigate, setShowNavigate] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        loadTabs();
    }, [instanceId]);

    async function loadTabs() {
        if (!instanceId) return;
        try {
            const list = await ListTabs(instanceId);
            setTabs(list || []);
            setError(null);
        } catch (err) {
            setError(String(err));
        }
    }

    async function createTab() {
        if (!instanceId) return;
        try {
            setLoading(true);
            setError(null);

            // Generate a random fingerprint for the new tab
            const fp = await GenerateRandomFingerprint('US');

            const tab = await CreateTab(instanceId, commands.TabConfig.createFrom({
                URL: url || 'about:blank',
                Fingerprint: fp,
                ProxyURL: '',
            }));

            setTabs(prev => [...prev, tab]);
            setSelectedTabId(tab.ID);
            setShowCreate(false);
            setUrl('');
            onTabCreated?.(tab);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    }

    async function closeTab(tabId: string, e: React.MouseEvent) {
        e.stopPropagation();
        if (!instanceId) return;
        try {
            setError(null);
            await CloseTab(instanceId, tabId);
            setTabs(prev => prev.filter(t => t.ID !== tabId));
            if (selectedTabId === tabId) {
                setSelectedTabId(tabs[0]?.ID || null);
            }
            onTabClosed?.(tabId);
        } catch (err) {
            setError(String(err));
        }
    }

    async function navigateToTab(tabId: string) {
        if (!instanceId || !navigateUrl) return;
        try {
            setError(null);
            await NavigateTab(instanceId, tabId, navigateUrl);
            setTabs(prev => prev.map(t =>
                t.ID === tabId ? commands.TabInfo.createFrom({ ...t, URL: navigateUrl }) : t
            ));
            setShowNavigate(false);
            setNavigateUrl('');
            onTabNavigated?.(tabId, navigateUrl);
        } catch (err) {
            setError(String(err));
        }
    }

    return (
        <div className="tab-fingerprint-selector">
            {error && (
                <div className="error-banner" style={{ color: 'red', padding: '8px', marginBottom: '8px' }}>
                    {error}
                </div>
            )}

            <div className="tab-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                <h3>Browser Tabs</h3>
                <button onClick={() => setShowCreate(true)} disabled={!instanceId}>
                    + New Tab
                </button>
            </div>

            <div className="tab-list" style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                {tabs.length === 0 ? (
                    <div style={{ color: '#666', padding: '16px', textAlign: 'center' }}>
                        No tabs. Click "+ New Tab" to create one.
                    </div>
                ) : (
                    tabs.map(tab => (
                        <div
                            key={tab.ID}
                            className={`tab-item ${selectedTabId === tab.ID ? 'selected' : ''}`}
                            style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: '8px',
                                padding: '8px 12px',
                                border: '1px solid #ddd',
                                borderRadius: '4px',
                                cursor: 'pointer',
                                background: selectedTabId === tab.ID ? '#e3f2fd' : '#fff',
                            }}
                            onClick={() => setSelectedTabId(tab.ID)}
                        >
                            <span className="tab-fp" style={{ fontFamily: 'monospace', fontSize: '11px', color: '#888' }}>
                                {tab.FingerprintSeed?.slice(0, 8) || 'N/A'}
                            </span>
                            <span className="tab-url" style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                {tab.URL || 'about:blank'}
                            </span>
                            <button
                                onClick={() => { setShowNavigate(true); setSelectedTabId(tab.ID); }}
                                style={{ padding: '2px 8px', fontSize: '11px' }}
                                title="Navigate"
                            >
                                →
                            </button>
                            <button
                                onClick={(e) => closeTab(tab.ID, e)}
                                style={{ padding: '2px 8px', fontSize: '11px', color: 'red' }}
                                title="Close"
                            >
                                ×
                            </button>
                        </div>
                    ))
                )}
            </div>

            {/* Create Tab Modal */}
            {showCreate && (
                <div className="modal-overlay" style={{
                    position: 'fixed',
                    top: 0, left: 0, right: 0, bottom: 0,
                    background: 'rgba(0,0,0,0.5)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    zIndex: 1000,
                }}>
                    <div className="modal" style={{
                        background: 'white',
                        padding: '24px',
                        borderRadius: '8px',
                        minWidth: '400px',
                    }}>
                        <h3 style={{ marginTop: 0 }}>Create New Tab</h3>
                        <div className="form-group" style={{ marginBottom: '16px' }}>
                            <label style={{ display: 'block', marginBottom: '4px' }}>URL</label>
                            <input
                                type="text"
                                placeholder="https://example.com"
                                value={url}
                                onChange={e => setUrl(e.target.value)}
                                style={{ width: '100%', padding: '8px', boxSizing: 'border-box' }}
                                onKeyDown={e => e.key === 'Enter' && createTab()}
                            />
                        </div>
                        <div style={{ color: '#666', fontSize: '12px', marginBottom: '16px' }}>
                            A new tab will be created with a random fingerprint assigned to it.
                        </div>
                        <div className="modal-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                            <button onClick={() => setShowCreate(false)}>Cancel</button>
                            <button onClick={createTab} disabled={loading}>
                                {loading ? 'Creating...' : 'Create'}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Navigate Tab Modal */}
            {showNavigate && selectedTabId && (
                <div className="modal-overlay" style={{
                    position: 'fixed',
                    top: 0, left: 0, right: 0, bottom: 0,
                    background: 'rgba(0,0,0,0.5)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    zIndex: 1000,
                }}>
                    <div className="modal" style={{
                        background: 'white',
                        padding: '24px',
                        borderRadius: '8px',
                        minWidth: '400px',
                    }}>
                        <h3 style={{ marginTop: 0 }}>Navigate Tab</h3>
                        <div className="form-group" style={{ marginBottom: '16px' }}>
                            <label style={{ display: 'block', marginBottom: '4px' }}>URL</label>
                            <input
                                type="text"
                                placeholder="https://example.com"
                                value={navigateUrl}
                                onChange={e => setNavigateUrl(e.target.value)}
                                style={{ width: '100%', padding: '8px', boxSizing: 'border-box' }}
                                onKeyDown={e => e.key === 'Enter' && navigateToTab(selectedTabId)}
                            />
                        </div>
                        <div className="modal-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                            <button onClick={() => { setShowNavigate(false); setNavigateUrl(''); }}>Cancel</button>
                            <button onClick={() => navigateToTab(selectedTabId)} disabled={!navigateUrl}>
                                Navigate
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
