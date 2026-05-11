import { useEffect, useState } from 'react';
import { InstancesPage } from './components/InstancesPage';
import { AccountsPage } from './components/AccountsPage';
import { ProxiesPage } from './components/ProxiesPage';
import { SettingsPage } from './components/SettingsPage';
import { TabsPage } from './components/TabsPage';
import { ListInstances, ListAccounts, ListProxies } from './wailsjs/go/main/App';
import { instance } from './wailsjs/go/models';

type DashboardProps = {
  onNewInstance: () => void;
  onAddAccount: () => void;
  onImportProxies: () => void;
};

function Dashboard({ onNewInstance, onAddAccount, onImportProxies }: DashboardProps) {
  const [runningCount, setRunningCount] = useState(0);
  const [accountCount, setAccountCount] = useState(0);
  const [proxyCount, setProxyCount] = useState(0);
  const [instanceStats, setInstanceStats] = useState<{ name: string; count: number }[]>([]);
  const [statsError, setStatsError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;

    const loadStats = async () => {
      try {
        const [list, accountList, proxyList] = await Promise.all([
          ListInstances(instance.InstanceFilter.createFrom({})),
          ListAccounts(),
          ListProxies(),
        ]);
        if (!mounted) return;
        const total = list || [];
        setRunningCount(total.filter(inst => inst.status === 'running').length);
        setAccountCount((accountList || []).length);
        setProxyCount((proxyList || []).length);
        const counts = new Map<string, number>();
        total.forEach(inst => {
          const name = inst.name || `${inst.id.slice(0, 8)}...`;
          counts.set(name, (counts.get(name) || 0) + 1);
        });
        setInstanceStats(Array.from(counts.entries()).map(([name, count]) => ({ name, count })));
        setStatsError(null);
      } catch (err) {
        if (!mounted) return;
        setStatsError(String(err));
      }
    };

    loadStats();
    const interval = setInterval(loadStats, 5000);
    return () => {
      mounted = false;
      clearInterval(interval);
    };
  }, []);

  return (
    <div className="dashboard">
      <h2>Dashboard</h2>
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{runningCount}</div>
          <div className="stat-label">Running Instances</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{accountCount}</div>
          <div className="stat-label">Total Accounts</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{proxyCount}</div>
          <div className="stat-label">Total Proxies</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">v1.0.0</div>
          <div className="stat-label">Version</div>
        </div>
      </div>
      <div className="dashboard-section">
        <h3>Quick Actions</h3>
        <div className="quick-actions">
          <button className="action-btn" onClick={onNewInstance}>New Instance</button>
          <button className="action-btn" onClick={onAddAccount}>Add Account</button>
          <button className="action-btn" onClick={onImportProxies}>Import Proxies</button>
        </div>
      </div>
      <div className="dashboard-section">
        <h3>Instance Names</h3>
        {statsError && <div className="error-banner">{statsError}</div>}
        {instanceStats.length === 0 ? (
          <div className="text-muted">No instances available.</div>
        ) : (
          <div className="instance-stats">
            {instanceStats.map(stat => (
              <div key={stat.name} className="instance-stat-row">
                <span>{stat.name}</span>
                <span className="instance-stat-count">{stat.count}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

type Tab = 'dashboard' | 'instances' | 'accounts' | 'proxies' | 'settings' | 'tabs';

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('dashboard');
  const [instanceCreateRequest, setInstanceCreateRequest] = useState(0);
  const [accountCreateRequest, setAccountCreateRequest] = useState(0);

  const handleNewInstance = () => {
    setActiveTab('instances');
    setInstanceCreateRequest(prev => prev + 1);
  };

  const handleAddAccount = () => {
    setActiveTab('accounts');
    setAccountCreateRequest(prev => prev + 1);
  };

  const handleImportProxies = () => {
    setActiveTab('proxies');
  };

  return (
    <div className="app">
      <nav className="sidebar">
        <div className="sidebar-header">
          <h1>fingerbrower</h1>
        </div>
        <ul className="nav-list">
          <li
            className={activeTab === 'dashboard' ? 'active' : ''}
            onClick={() => setActiveTab('dashboard')}
          >
            <span className="nav-icon">◈</span>
            Dashboard
          </li>
          <li
            className={activeTab === 'instances' ? 'active' : ''}
            onClick={() => setActiveTab('instances')}
          >
            <span className="nav-icon">◎</span>
            Instances
          </li>
          <li
            className={activeTab === 'accounts' ? 'active' : ''}
            onClick={() => setActiveTab('accounts')}
          >
            <span className="nav-icon">◉</span>
            Accounts
          </li>
          <li
            className={activeTab === 'proxies' ? 'active' : ''}
            onClick={() => setActiveTab('proxies')}
          >
            <span className="nav-icon">⬢</span>
            Proxies
          </li>
          <li
            className={activeTab === 'tabs' ? 'active' : ''}
            onClick={() => setActiveTab('tabs')}
          >
            <span className="nav-icon">⧉</span>
            Tabs
          </li>
          <li
            className={activeTab === 'settings' ? 'active' : ''}
            onClick={() => setActiveTab('settings')}
          >
            <span className="nav-icon">⚙</span>
            Settings
          </li>
        </ul>
      </nav>
      <main className="content">
        {activeTab === 'dashboard' && (
          <Dashboard
            onNewInstance={handleNewInstance}
            onAddAccount={handleAddAccount}
            onImportProxies={handleImportProxies}
          />
        )}
        {activeTab === 'instances' && (
          <InstancesPage createRequest={instanceCreateRequest} />
        )}
        {activeTab === 'accounts' && (
          <AccountsPage createRequest={accountCreateRequest} />
        )}
        {activeTab === 'proxies' && <ProxiesPage />}
        {activeTab === 'tabs' && <TabsPage />}
        {activeTab === 'settings' && <SettingsPage />}
      </main>
    </div>
  );
}

export default App;
