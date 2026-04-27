import { useState } from 'react';
import { InstancesPage } from './components/InstancesPage';
import { AccountsPage } from './components/AccountsPage';
import { SettingsPage } from './components/SettingsPage';

function Dashboard() {
  return (
    <div className="dashboard">
      <h2>Dashboard</h2>
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">0</div>
          <div className="stat-label">Running Instances</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">0</div>
          <div className="stat-label">Total Accounts</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">0</div>
          <div className="stat-label">Active Proxies</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">v1.0.0</div>
          <div className="stat-label">Version</div>
        </div>
      </div>
      <div className="dashboard-section">
        <h3>Quick Actions</h3>
        <div className="quick-actions">
          <button className="action-btn">New Instance</button>
          <button className="action-btn">Add Account</button>
          <button className="action-btn">Import Proxies</button>
        </div>
      </div>
    </div>
  );
}

type Tab = 'dashboard' | 'instances' | 'accounts' | 'settings';

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('dashboard');

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
            className={activeTab === 'settings' ? 'active' : ''}
            onClick={() => setActiveTab('settings')}
          >
            <span className="nav-icon">⚙</span>
            Settings
          </li>
        </ul>
      </nav>
      <main className="content">
        {activeTab === 'dashboard' && <Dashboard />}
        {activeTab === 'instances' && <InstancesPage />}
        {activeTab === 'accounts' && <AccountsPage />}
        {activeTab === 'settings' && <SettingsPage />}
      </main>
    </div>
  );
}

export default App;
