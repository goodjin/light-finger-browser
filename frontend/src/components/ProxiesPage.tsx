import { useEffect, useState } from 'react';
import {
  ListProxies,
  CreateProxy,
  DeleteProxy,
  ListInstances,
} from '../wailsjs/go/main/App';
import { commands, instance } from '../wailsjs/go/models';

export function ProxiesPage() {
  const [proxies, setProxies] = useState<commands.ProxyDTO[]>([]);
  const [instances, setInstances] = useState<commands.BrowserInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [pendingDeleteID, setPendingDeleteID] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [url, setURL] = useState('');
  const [country, setCountry] = useState('');
  const [proxyType, setProxyType] = useState('residential');
  const [provider, setProvider] = useState('');

  useEffect(() => {
    loadAll();
    const interval = setInterval(loadAll, 5000);
    return () => clearInterval(interval);
  }, []);

  async function loadAll() {
    try {
      const [proxyList, instanceList] = await Promise.all([
        ListProxies(),
        ListInstances(instance.InstanceFilter.createFrom({})),
      ]);
      setProxies(proxyList || []);
      setInstances(instanceList || []);
      setError(null);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  function isProxyInUse(proxyID: string) {
    return instances.some(inst => inst.proxy_id === proxyID && inst.status !== 'stopped');
  }

  async function createProxy() {
    try {
      if (creating) {
        return;
      }
      setCreating(true);
      setError(null);
      const created = await CreateProxy({
        url,
        country,
        type: proxyType,
        provider,
      });
      setProxies([created, ...proxies]);
      setShowCreate(false);
      setURL('');
      setCountry('');
      setProvider('');
      setProxyType('residential');
    } catch (err) {
      setError(String(err));
    } finally {
      setCreating(false);
    }
  }

  async function deleteProxy(id: string): Promise<boolean> {
    try {
      await DeleteProxy(id);
      setProxies(proxies.filter(p => p.id !== id));
      return true;
    } catch (err) {
      setError(String(err));
      return false;
    }
  }

  return (
    <div className="proxies-page">
      <div className="page-header">
        <h2>Proxies</h2>
        <button className="btn-primary" onClick={() => setShowCreate(true)}>
          + Add Proxy
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {showCreate && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Add Proxy</h3>
            <div className="form-group">
              <label>Proxy URL</label>
              <input
                type="text"
                value={url}
                disabled={creating}
                onChange={e => setURL(e.target.value)}
                placeholder="user:pass@host:port"
              />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Country (optional)</label>
                <input
                type="text"
                value={country}
                disabled={creating}
                onChange={e => setCountry(e.target.value)}
                placeholder="US"
              />
              </div>
              <div className="form-group">
                <label>Type</label>
                <select
                  value={proxyType}
                  disabled={creating}
                  onChange={e => setProxyType(e.target.value)}
                >
                  <option value="residential">Residential</option>
                  <option value="datacenter">Datacenter</option>
                </select>
              </div>
            </div>
            <div className="form-group">
              <label>Provider (optional)</label>
              <input
                type="text"
                value={provider}
                disabled={creating}
                onChange={e => setProvider(e.target.value)}
                placeholder="manual"
              />
            </div>
            <div className="modal-actions">
              <button disabled={creating} onClick={() => {
                setShowCreate(false);
                setURL('');
                setCountry('');
                setProvider('');
                setProxyType('residential');
              }}>Cancel</button>
              <button className="btn-primary" onClick={createProxy} disabled={creating}>
                {creating ? 'Adding...' : 'Add'}
              </button>
            </div>
          </div>
        </div>
      )}

      {pendingDeleteID && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Delete Proxy</h3>
            <p>Stop any running instances using this proxy before deleting.</p>
            <div className="modal-actions">
              <button onClick={() => setPendingDeleteID(null)} disabled={deletingId === pendingDeleteID}>
                Cancel
              </button>
              <button
                className="btn-danger"
                onClick={async () => {
                  const id = pendingDeleteID;
                  if (!id || deletingId) {
                    return;
                  }
                  setDeletingId(id);
                  const ok = await deleteProxy(id);
                  if (ok) {
                    setPendingDeleteID(null);
                  }
                  setDeletingId(null);
                }}
                disabled={deletingId === pendingDeleteID}
              >
                {deletingId === pendingDeleteID ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading">Loading proxies...</div>
      ) : proxies.length === 0 ? (
        <div className="empty-state">
          <p>No proxies yet. Add your first proxy.</p>
        </div>
      ) : (
        <div className="accounts-list">
          {proxies.map(proxy => (
            <div key={proxy.id} className="account-card">
              <div className="account-info">
                <strong>{proxy.url}</strong>
                <span className="account-status">{proxy.status}</span>
              </div>
              <div className="account-meta">
                Country: {proxy.country || '-'} | Type: {proxy.type || '-'} | Provider: {proxy.provider || '-'}
              </div>
              {isProxyInUse(proxy.id) && (
                <div className="warning-banner">In use by running instance</div>
              )}
              <div className="account-actions">
                <button
                  className="btn-secondary"
                  onClick={() => setPendingDeleteID(proxy.id)}
                  disabled={deletingId === proxy.id || isProxyInUse(proxy.id)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
