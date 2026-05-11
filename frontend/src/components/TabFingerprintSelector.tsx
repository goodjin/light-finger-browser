import { useState, useEffect } from 'react';
import {
    CreateTab,
    CloseTab,
    ListTabs,
    NavigateTab,
    GenerateRandomFingerprint,
} from '../wailsjs/go/main/App';
import { commands } from '../wailsjs/go/models';

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
    const [selectedCountry, setSelectedCountry] = useState<string>('US');
    const [showAdvancedConfig, setShowAdvancedConfig] = useState(false);
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

            // Generate a random fingerprint with the selected country
            const fp = await GenerateRandomFingerprint(selectedCountry);

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
                        minWidth: '450px',
                    }}>
                        <h3 style={{ marginTop: 0 }}>Create New Tab</h3>
                        <div className="form-group" style={{ marginBottom: '16px' }}>
                            <label style={{ display: 'block', marginBottom: '4px', fontWeight: 500 }}>URL (optional)</label>
                            <input
                                type="text"
                                placeholder="https://example.com"
                                value={url}
                                onChange={e => setUrl(e.target.value)}
                                style={{ width: '100%', padding: '8px', boxSizing: 'border-box', border: '1px solid #ddd', borderRadius: '4px' }}
                                onKeyDown={e => e.key === 'Enter' && createTab()}
                            />
                        </div>
                        
                        {/* Fingerprint Configuration */}
                        <div className="form-group" style={{ marginBottom: '16px' }}>
                            <label style={{ display: 'block', marginBottom: '4px', fontWeight: 500 }}>
                                Fingerprint Country
                            </label>
                            <select
                                value={selectedCountry}
                                onChange={e => setSelectedCountry(e.target.value)}
                                disabled={loading}
                                style={{
                                    width: '100%',
                                    padding: '8px',
                                    boxSizing: 'border-box',
                                    border: '1px solid #ddd',
                                    borderRadius: '4px',
                                    fontSize: '14px',
                                    background: '#fff',
                                }}
                            >
                                {SUPPORTED_COUNTRIES.map(country => (
                                    <option key={country.code} value={country.code}>
                                        {country.flag} {country.name} ({country.code})
                                    </option>
                                ))}
                            </select>
                            <div style={{ color: '#666', fontSize: '12px', marginTop: '4px' }}>
                                Each tab will have a unique fingerprint based on the selected country.
                            </div>
                        </div>

                        {/* Advanced Configuration Toggle */}
                        <div style={{ marginBottom: '16px' }}>
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
                                    fontSize: '10px',
                                }}>
                                    ▶
                                </span>
                                Advanced Configuration
                            </button>
                            
                            {showAdvancedConfig && (
                                <div style={{
                                    marginTop: '8px',
                                    padding: '10px',
                                    background: '#f9fafb',
                                    borderRadius: '4px',
                                    border: '1px solid #e5e7eb',
                                    fontSize: '12px',
                                }}>
                                    <div style={{ marginBottom: '6px', color: '#666' }}>
                                        Settings for <strong>{SUPPORTED_COUNTRIES.find(c => c.code === selectedCountry)?.name}</strong>:
                                    </div>
                                    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                            <span style={{ color: '#888' }}>Timezone:</span>
                                            <span style={{ fontFamily: 'monospace' }}>{SUPPORTED_COUNTRIES.find(c => c.code === selectedCountry)?.timezone}</span>
                                        </div>
                                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                            <span style={{ color: '#888' }}>Platform:</span>
                                            <span style={{ fontFamily: 'monospace' }}>Windows</span>
                                        </div>
                                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                            <span style={{ color: '#888' }}>Locale:</span>
                                            <span style={{ fontFamily: 'monospace' }}>{SUPPORTED_COUNTRIES.find(c => c.code === selectedCountry)?.code.toLowerCase() + '-' + SUPPORTED_COUNTRIES.find(c => c.code === selectedCountry)?.code.toUpperCase()}</span>
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>

                        <div className="modal-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                            <button 
                                onClick={() => {
                                    setShowCreate(false);
                                    setUrl('');
                                    setShowAdvancedConfig(false);
                                }}
                                disabled={loading}
                                style={{ padding: '8px 16px' }}
                            >
                                Cancel
                            </button>
                            <button 
                                onClick={createTab} 
                                disabled={loading}
                                style={{ 
                                    padding: '8px 16px',
                                    background: '#1976d2',
                                    color: 'white',
                                    border: 'none',
                                    borderRadius: '4px',
                                    cursor: 'pointer',
                                }}
                            >
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
