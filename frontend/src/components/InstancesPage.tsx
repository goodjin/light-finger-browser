import { useState, useEffect } from 'react';
import {
  CreateInstance,
  StopInstance,
  RestartInstance,
  DeleteInstance,
  ListInstances,
  ListAccounts,
  ListProxies,
  GenerateRandomFingerprint,
  CheckFingerprint,
  BindAccountInstance,
} from '../wailsjs/go/main/App';
import { commands, instance } from '../wailsjs/go/models';

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

type InstancesPageProps = {
  createRequest: number;
};

export function InstancesPage({ createRequest }: InstancesPageProps) {
  const [instances, setInstances] = useState<commands.BrowserInstance[]>([]);
  const [accounts, setAccounts] = useState<commands.Account[]>([]);
  const [proxies, setProxies] = useState<commands.ProxyDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [pendingStopID, setPendingStopID] = useState<string | null>(null);
  const [pendingDeleteID, setPendingDeleteID] = useState<string | null>(null);
  const [stoppingId, setStoppingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [restartingId, setRestartingId] = useState<string | null>(null);
  const [checkingId, setCheckingId] = useState<string | null>(null);
  const [fingerprintResult, setFingerprintResult] = useState<commands.FingerprintCheckResult | null>(null);
  const [fingerprintError, setFingerprintError] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [nameTouched, setNameTouched] = useState(false);
  const [country, setCountry] = useState('US');
  const [group, setGroup] = useState('');
  const [groupTouched, setGroupTouched] = useState(false);
  const [headless, setHeadless] = useState(false);
  const [headlessTouched, setHeadlessTouched] = useState(false);
  const [selectedAccountID, setSelectedAccountID] = useState('');
  const [selectedProxyID, setSelectedProxyID] = useState('');

  useEffect(() => {
    loadAll();
    const interval = setInterval(loadAll, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (createRequest > 0) {
      setShowCreate(true);
    }
  }, [createRequest]);

  const selectedAccount = accounts.find(acc => acc.id === selectedAccountID);

  useEffect(() => {
    if (!selectedAccount) {
      return;
    }
    if (!nameTouched) {
      setName(selectedAccount.instance_name || selectedAccount.label || '');
    }
    if (!groupTouched) {
      setGroup(selectedAccount.group || '');
    }
    if (!headlessTouched) {
      setHeadless(!!selectedAccount.headless);
    }
  }, [selectedAccountID, selectedAccount, nameTouched, groupTouched, headlessTouched]);

  async function loadAll() {
    try {
      const [list, accountList, proxyList] = await Promise.all([
        ListInstances(instance.InstanceFilter.createFrom({})),
        ListAccounts(),
        ListProxies(),
      ]);
      setInstances(list || []);
      setAccounts(accountList || []);
      setProxies(proxyList || []);
      setError(null);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function createInstance() {
    try {
      if (creating) {
        return;
      }
      setCreating(true);
      setLoading(true);
      setError(null);
      if (selectedAccountID) {
        if (selectedAccount?.instance_status && selectedAccount.instance_status !== 'stopped') {
          throw new Error('Selected account is already bound to a running instance. Stop it before creating a new one.');
        }
        const selectedProxy = proxies.find(p => p.id === selectedProxyID);
        await BindAccountInstance({
          account_id: selectedAccountID,
          instance_name: name,
          group,
          headless,
          proxy_id: selectedProxyID,
          proxy_url: selectedProxy?.url || '',
        });
      } else {
        const fp = await GenerateRandomFingerprint(country);
        const selectedProxy = proxies.find(p => p.id === selectedProxyID);
        const cfg = new instance.InstanceConfig({
          name,
          fingerprint: fp,
          account_id: '',
          group,
          headless,
          proxy: selectedProxy
            ? new instance.ProxyConfig({
              id: selectedProxy.id,
              url: selectedProxy.url,
            })
            : undefined,
          });
        await CreateInstance(cfg);
      }
      setShowCreate(false);
      resetCreateForm();
      await loadAll();
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
      setCreating(false);
    }
  }

  async function stopInstance(id: string): Promise<boolean> {
    try {
      await StopInstance(id);
      await loadAll();
      return true;
    } catch (err) {
      setError(String(err));
      return false;
    }
  }

  async function restartInstance(id: string) {
    try {
      if (restartingId === id) {
        return;
      }
      setRestartingId(id);
      await RestartInstance(id);
      await loadAll();
    } catch (err) {
      setError(String(err));
    } finally {
      setRestartingId(null);
    }
  }

  async function deleteInstance(id: string): Promise<boolean> {
    try {
      await DeleteInstance(id);
      await loadAll();
      return true;
    } catch (err) {
      setError(String(err));
      return false;
    }
  }

  async function checkFingerprint(id: string) {
    try {
      if (checkingId === id) {
        return;
      }
      setCheckingId(id);
      setFingerprintError(null);
      const result = await CheckFingerprint(id);
      setFingerprintResult(result || null);
    } catch (err) {
      setFingerprintError(String(err));
    } finally {
      setCheckingId(null);
    }
  }

  function resetCreateForm() {
    setName('');
    setNameTouched(false);
    setGroup('');
    setGroupTouched(false);
    setHeadless(false);
    setHeadlessTouched(false);
    setSelectedAccountID('');
    setSelectedProxyID('');
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
              <label>Name</label>
              <input
                type="text"
                value={name}
                disabled={creating}
                onChange={e => {
                  setNameTouched(true);
                  setName(e.target.value);
                }}
                placeholder="e.g., account-a"
              />
            </div>
            <div className="form-group">
              <label>Account (optional)</label>
              <select
                value={selectedAccountID}
                disabled={creating}
                onChange={e => setSelectedAccountID(e.target.value)}
              >
                <option value="">No account</option>
                {accounts.map(acc => (
                  <option key={acc.id} value={acc.id}>
                    {acc.label || acc.email || acc.id.slice(0, 8)}
                    {acc.instance_status && acc.instance_status !== 'stopped' ? ' (In use)' : ''}
                  </option>
                ))}
              </select>
              {selectedAccount && (
                <div className="helper-text">
                  Uses the selected account fingerprint and proxy defaults.
                </div>
              )}
            </div>
            <div className="form-group">
              <label>Country (for fingerprint)</label>
              <select
                value={country}
                disabled={creating || !!selectedAccountID}
                onChange={e => setCountry(e.target.value)}
              >
                {COUNTRIES.map(c => (
                  <option key={c.code} value={c.code}>{c.name}</option>
                ))}
              </select>
              {selectedAccountID && (
                <div className="helper-text">Fingerprint settings come from the account.</div>
              )}
            </div>
            <div className="form-group">
              <label>Group</label>
              <input
                type="text"
                value={group}
                disabled={creating}
                onChange={e => {
                  setGroupTouched(true);
                  setGroup(e.target.value);
                }}
                placeholder="e.g., marketing, testing"
              />
            </div>
            <div className="form-group">
              <label>Proxy (optional)</label>
              <select
                value={selectedProxyID}
                disabled={creating}
                onChange={e => setSelectedProxyID(e.target.value)}
              >
                <option value="">No proxy</option>
                {proxies.map(p => (
                  <option key={p.id} value={p.id}>
                    {p.url}
                    {instances.some(inst => inst.proxy_id === p.id && inst.status !== 'stopped') ? ' (In use)' : ''}
                  </option>
                ))}
              </select>
              {selectedAccount && !selectedProxyID && selectedAccount.proxy_url && (
                <div className="helper-text">
                  Account proxy will be used if none is selected.
                </div>
              )}
            </div>
            <div className="form-group">
              <label>
                <input
                  type="checkbox"
                    checked={headless}
                    disabled={creating}
                    onChange={e => {
                      setHeadlessTouched(true);
                      setHeadless(e.target.checked);
                  }}
                />
                Headless mode
              </label>
            </div>
            <div className="modal-actions">
              <button disabled={creating} onClick={() => {
                setShowCreate(false);
                resetCreateForm();
              }}>Cancel</button>
              <button className="btn-primary" onClick={createInstance} disabled={creating}>
                {creating ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {pendingStopID && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Stop Instance</h3>
            <p>This will terminate the running browser process.</p>
            <div className="modal-actions">
                <button onClick={() => setPendingStopID(null)} disabled={stoppingId === pendingStopID}>
                  Cancel
                </button>
                <button
                  className="btn-danger"
                  onClick={async () => {
                    const id = pendingStopID;
                    if (!id || stoppingId) {
                      return;
                    }
                    setStoppingId(id);
                    const ok = await stopInstance(id);
                    if (ok) {
                      setPendingStopID(null);
                    }
                    setStoppingId(null);
                  }}
                  disabled={stoppingId === pendingStopID}
                >
                  {stoppingId === pendingStopID ? 'Stopping...' : 'Stop'}
                </button>
              </div>
            </div>
          </div>
        )}

      {pendingDeleteID && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Delete Instance</h3>
            <p>This will remove the instance record and its user data directory.</p>
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
                    const ok = await deleteInstance(id);
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

      {fingerprintResult && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Fingerprint Check</h3>
            {fingerprintError && <div className="error-banner">{fingerprintError}</div>}
            <div className="form-group">
              <label>Status</label>
              <div>{fingerprintResult.matches ? 'Match' : 'Mismatch'}</div>
            </div>
            {fingerprintResult.diffs?.length > 0 && (
              <div className="form-group">
                <label>Differences</label>
                <div>{fingerprintResult.diffs.join(', ')}</div>
              </div>
            )}
            {fingerprintResult.coverage_gaps?.length > 0 && (
              <div className="form-group">
                <label>Coverage Gaps</label>
                <div>{fingerprintResult.coverage_gaps.join(', ')}</div>
              </div>
            )}
            {fingerprintResult.expected && (
              <div className="form-group">
                <label>Expected Snapshot</label>
                <pre className="code-block">
                  {JSON.stringify(fingerprintResult.expected, null, 2)}
                </pre>
              </div>
            )}
            <div className="form-group">
              <label>Snapshot</label>
              <pre className="code-block">
                {JSON.stringify(fingerprintResult.snapshot, null, 2)}
              </pre>
            </div>
            {fingerprintResult.previous && (
              <div className="form-group">
                <label>Previous Snapshot</label>
                <pre className="code-block">
                  {JSON.stringify(fingerprintResult.previous, null, 2)}
                </pre>
              </div>
            )}
            <div className="modal-actions">
              <button onClick={() => setFingerprintResult(null)}>Close</button>
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
                <span className="instance-id">{inst.name || `${inst.id.slice(0, 8)}...`}</span>
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
                <div>Account: {inst.account_label || '-'}</div>
                <div>Proxy: {inst.proxy_url || '-'}</div>
                <div>
                  Fingerprint: {inst.fingerprint?.platform || 'N/A'}
                </div>
              </div>
              <div className="instance-actions">
                <button
                  className="btn-danger"
                  onClick={() => setPendingStopID(inst.id)}
                  disabled={stoppingId === inst.id || inst.status === 'stopping' || inst.status === 'stopped'}
                >
                  {stoppingId === inst.id ? 'Stopping...' : 'Stop'}
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => restartInstance(inst.id)}
                  disabled={restartingId === inst.id || inst.status !== 'stopped'}
                >
                  {restartingId === inst.id ? 'Restarting...' : 'Restart'}
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => setPendingDeleteID(inst.id)}
                  disabled={deletingId === inst.id || inst.status !== 'stopped'}
                >
                  {deletingId === inst.id ? 'Deleting...' : 'Delete'}
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => checkFingerprint(inst.id)}
                  disabled={checkingId === inst.id}
                >
                  {checkingId === inst.id ? 'Checking...' : 'Fingerprint'}
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
