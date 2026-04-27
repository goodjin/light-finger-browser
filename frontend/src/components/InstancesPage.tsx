import { useState, useEffect } from 'react';
import {
  CreateInstance,
  DestroyInstance,
  ListInstances,
  GenerateRandomFingerprint,
  type BrowserInstance,
  type InstanceConfig,
} from '../wailsjs/go/main';

const COUNTRIES = [
  { code: 'US', name: 'United States' },
  { code: 'GB', name: 'United Kingdom' },
  { code: 'DE', name: 'Germany' },
  { code: 'FR', name: 'France' },
  { code: 'JP', name: 'Japan' },
  { code: 'CN', name: 'China' },
  { code: 'CA', name: 'Canada' },
  { code: 'AU', name: 'Australia' },
  { code: 'BR', name: 'Brazil' },
  { code: 'IN', name: 'India' },
];

export function InstancesPage() {
  const [instances, setInstances] = useState<BrowserInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [country, setCountry] = useState('US');
  const [group, setGroup] = useState('');
  const [headless, setHeadless] = useState(false);

  useEffect(() => {
    loadInstances();
    const interval = setInterval(loadInstances, 5000);
    return () => clearInterval(interval);
  }, []);

  async function loadInstances() {
    try {
      const list = await ListInstances(null);
      setInstances(list || []);
      setError(null);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function createInstance() {
    try {
      setLoading(true);
      const fp = await GenerateRandomFingerprint(country);
      const cfg: InstanceConfig = {
        fingerprint: fp,
        account_id: '',
        group,
        headless,
      };
      await CreateInstance(cfg);
      await loadInstances();
      setShowCreate(false);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function destroyInstance(id: string) {
    if (!confirm('Are you sure you want to stop this instance?')) return;
    try {
      await DestroyInstance(id);
      await loadInstances();
    } catch (err) {
      setError(String(err));
    }
  }

  function getStatusColor(status: string): string {
    switch (status) {
      case 'running': return '#22c55e';
      case 'starting': return '#eab308';
      case 'stopping': return '#f97316';
      case 'error': return '#ef4444';
      default: return '#6b7280';
    }
  }

  if (loading && instances.length === 0) {
    return <div className="loading">Loading instances...</div>;
  }

  return (
    <div className="instances-page">
      <div className="page-header">
        <h2>Browser Instances</h2>
        <button className="btn-primary" onClick={() => setShowCreate(true)}>
          + New Instance
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {showCreate && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Create New Instance</h3>
            <div className="form-group">
              <label>Country (for fingerprint)</label>
              <select value={country} onChange={e => setCountry(e.target.value)}>
                {COUNTRIES.map(c => (
                  <option key={c.code} value={c.code}>{c.name}</option>
                ))}
              </select>
            </div>
            <div className="form-group">
              <label>Group</label>
              <input
                type="text"
                value={group}
                onChange={e => setGroup(e.target.value)}
                placeholder="e.g., marketing, testing"
              />
            </div>
            <div className="form-group">
              <label>
                <input
                  type="checkbox"
                  checked={headless}
                  onChange={e => setHeadless(e.target.checked)}
                />
                Headless mode
              </label>
            </div>
            <div className="modal-actions">
              <button onClick={() => setShowCreate(false)}>Cancel</button>
              <button className="btn-primary" onClick={createInstance}>Create</button>
            </div>
          </div>
        </div>
      )}

      <div className="instances-grid">
        {instances.length === 0 ? (
          <div className="empty-state">
            <p>No instances yet. Create your first browser instance.</p>
          </div>
        ) : (
          instances.map(inst => (
            <div key={inst.id} className="instance-card">
              <div className="instance-header">
                <span className="instance-id">{inst.id.slice(0, 8)}...</span>
                <span
                  className="instance-status"
                  style={{ backgroundColor: getStatusColor(inst.status) }}
                >
                  {inst.status}
                </span>
              </div>
              <div className="instance-details">
                <div>Port: {inst.port}</div>
                <div>Group: {inst.group || '-'}</div>
                <div>
                  Fingerprint: {inst.fingerprint?.platform || 'N/A'}
                </div>
              </div>
              <div className="instance-actions">
                <button
                  className="btn-danger"
                  onClick={() => destroyInstance(inst.id)}
                  disabled={inst.status === 'stopping' || inst.status === 'stopped'}
                >
                  Stop
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
