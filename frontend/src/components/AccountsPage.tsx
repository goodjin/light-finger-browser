import { useState, useEffect } from 'react';
import {
  ListAccounts,
  CreateAccount,
  UpdateAccount,
  RestartAccountInstance,
  DeleteAccount,
} from '../wailsjs/go/main/App';
import { commands } from '../wailsjs/go/models';

type AccountsPageProps = {
  createRequest: number;
};

export function AccountsPage({ createRequest }: AccountsPageProps) {
  const [accounts, setAccounts] = useState<commands.Account[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [editingAccount, setEditingAccount] = useState<commands.Account | null>(null);
  const [pendingUpdate, setPendingUpdate] = useState<commands.AccountUpdateRequest | null>(null);
  const [showRestartPrompt, setShowRestartPrompt] = useState(false);
  const [pendingDeleteID, setPendingDeleteID] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Form state
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [group, setGroup] = useState('');
  const [instanceName, setInstanceName] = useState('');
  const [instanceNameTouched, setInstanceNameTouched] = useState(false);
  const [proxyURL, setProxyURL] = useState('');
  const [fingerprintSeed, setFingerprintSeed] = useState('');
  const [fingerprintCountry, setFingerprintCountry] = useState('US');
  const [headless, setHeadless] = useState(false);

  useEffect(() => {
    loadAccounts();
  }, []);

  useEffect(() => {
    if (createRequest > 0) {
      setShowCreate(true);
    }
  }, [createRequest]);

  useEffect(() => {
    if (!instanceNameTouched) {
      if (username) {
        setInstanceName(username);
      } else if (email) {
        setInstanceName(email);
      }
    }
  }, [username, email, instanceNameTouched]);

  async function loadAccounts() {
    setLoading(true);
    try {
      const list = await ListAccounts();
      setAccounts(list || []);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function createAccount() {
    try {
      const created = await CreateAccount({
        email,
        username,
        group,
        instance_name: instanceName,
        proxy_url: proxyURL,
        fingerprint_seed: fingerprintSeed,
        fingerprint_country: fingerprintCountry,
        headless,
      });
      setAccounts([created, ...accounts]);
      resetForm();
      setShowCreate(false);
    } catch (err) {
      setError(String(err));
    }
  }

  function resetForm() {
    setUsername('');
    setEmail('');
    setGroup('');
    setInstanceName('');
    setInstanceNameTouched(false);
    setProxyURL('');
    setFingerprintSeed('');
    setFingerprintCountry('US');
    setHeadless(false);
  }

  function startEdit(account: commands.Account) {
    setEditingAccount(account);
    setUsername(account.username || '');
    setEmail(account.email || '');
    setGroup(account.group || '');
    setInstanceName(account.instance_name || '');
    setInstanceNameTouched(false);
    setProxyURL(account.proxy_url || '');
    setFingerprintSeed(account.fingerprint_seed || '');
    setFingerprintCountry(account.fingerprint_country || 'US');
    setHeadless(!!account.headless);
  }

  async function saveAccount() {
    if (!editingAccount) return;

    const update: commands.AccountUpdateRequest = {
      id: editingAccount.id,
      email,
      username,
      group,
      instance_name: instanceName,
      proxy_url: proxyURL,
      fingerprint_seed: fingerprintSeed,
      fingerprint_country: fingerprintCountry,
      headless,
      restart: false,
    };

    const requiresRestart = (
      editingAccount.proxy_url !== proxyURL ||
      editingAccount.fingerprint_seed !== fingerprintSeed ||
      editingAccount.fingerprint_country !== fingerprintCountry ||
      editingAccount.headless !== headless
    );

    if (requiresRestart) {
      setPendingUpdate(update);
      setShowRestartPrompt(true);
      return;
    }

    await applyUpdate(update);
  }

  async function applyUpdate(update: commands.AccountUpdateRequest) {
    try {
      const updated = await UpdateAccount(update);
      setAccounts(accounts.map(acc => acc.id === updated.id ? updated : acc));
      setEditingAccount(null);
      resetForm();
    } catch (err) {
      setError(String(err));
    }
  }

  async function restartInstance(account: commands.Account) {
    try {
      const updated = await RestartAccountInstance(account.id);
      setAccounts(accounts.map(acc => acc.id === updated.id ? updated : acc));
    } catch (err) {
      setError(String(err));
    }
  }

  async function deleteAccount(id: string) {
    try {
      await DeleteAccount(id);
      setAccounts(accounts.filter(acc => acc.id !== id));
    } catch (err) {
      setError(String(err));
    }
  }

  return (
    <div className="accounts-page">
      <div className="page-header">
        <h2>Accounts</h2>
        <button className="btn-primary" onClick={() => setShowCreate(true)}>
          + Add Account
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {showCreate && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Add Account</h3>
            <p className="section-desc">
              All fields are optional. Changing proxy or fingerprint later may require a restart.
            </p>
            <div className="form-group">
              <label>Username (optional)</label>
              <input
                type="text"
                value={username}
                onChange={e => setUsername(e.target.value)}
                placeholder="Account username"
              />
            </div>
            <div className="form-group">
              <label>Email (optional)</label>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="account@example.com"
              />
            </div>
            <div className="form-group">
              <label>Group (optional)</label>
              <input
                type="text"
                value={group}
                onChange={e => setGroup(e.target.value)}
                placeholder="e.g., growth"
              />
            </div>
            <div className="form-group">
              <label>Instance Name (optional)</label>
              <input
                type="text"
                value={instanceName}
                onChange={e => {
                  setInstanceNameTouched(true);
                  setInstanceName(e.target.value);
                }}
                placeholder="default: account name"
              />
            </div>
            <div className="form-group">
              <label>Proxy URL (optional)</label>
              <input
                type="text"
                value={proxyURL}
                onChange={e => setProxyURL(e.target.value)}
                placeholder="user:pass@host:port"
              />
            </div>
            <div className="form-group">
              <label>Fingerprint Seed (optional)</label>
              <input
                type="text"
                value={fingerprintSeed}
                onChange={e => setFingerprintSeed(e.target.value)}
                placeholder="leave blank to auto-generate"
              />
            </div>
            <div className="form-group">
              <label>Fingerprint Country</label>
              <select value={fingerprintCountry} onChange={e => setFingerprintCountry(e.target.value)}>
                <option value="US">United States</option>
                <option value="GB">United Kingdom</option>
                <option value="DE">Germany</option>
                <option value="FR">France</option>
                <option value="JP">Japan</option>
                <option value="CN">China</option>
                <option value="CA">Canada</option>
                <option value="AU">Australia</option>
                <option value="BR">Brazil</option>
                <option value="IN">India</option>
              </select>
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
              <button onClick={() => {
                setShowCreate(false);
                resetForm();
              }}>Cancel</button>
              <button className="btn-primary" onClick={createAccount}>Add</button>
            </div>
          </div>
        </div>
      )}

      {editingAccount && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Edit Account</h3>
            <div className="form-group">
              <label>Username</label>
              <input
                type="text"
                value={username}
                onChange={e => setUsername(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label>Email</label>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label>Group</label>
              <input
                type="text"
                value={group}
                onChange={e => setGroup(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label>Instance Name</label>
              <input
                type="text"
                value={instanceName}
                onChange={e => {
                  setInstanceNameTouched(true);
                  setInstanceName(e.target.value);
                }}
              />
            </div>
            <div className="form-group">
              <label>Proxy URL</label>
              <input
                type="text"
                value={proxyURL}
                onChange={e => setProxyURL(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label>Fingerprint Seed</label>
              <input
                type="text"
                value={fingerprintSeed}
                onChange={e => setFingerprintSeed(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label>Fingerprint Country</label>
              <select value={fingerprintCountry} onChange={e => setFingerprintCountry(e.target.value)}>
                <option value="US">United States</option>
                <option value="GB">United Kingdom</option>
                <option value="DE">Germany</option>
                <option value="FR">France</option>
                <option value="JP">Japan</option>
                <option value="CN">China</option>
                <option value="CA">Canada</option>
                <option value="AU">Australia</option>
                <option value="BR">Brazil</option>
                <option value="IN">India</option>
              </select>
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
              <button onClick={() => {
                setEditingAccount(null);
                resetForm();
              }}>Cancel</button>
              <button className="btn-primary" onClick={saveAccount}>Save</button>
            </div>
          </div>
        </div>
      )}

      {showRestartPrompt && pendingUpdate && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Restart Required</h3>
            <p>Proxy or fingerprint changes require restarting the instance to take effect.</p>
            <div className="modal-actions">
              <button onClick={() => {
                setShowRestartPrompt(false);
                setPendingUpdate(null);
              }}>Cancel</button>
              <button onClick={async () => {
                const update = pendingUpdate;
                update.restart = false;
                setShowRestartPrompt(false);
                setPendingUpdate(null);
                await applyUpdate(update);
              }}>Save without restart</button>
              <button className="btn-primary" onClick={async () => {
                const update = pendingUpdate;
                update.restart = true;
                setShowRestartPrompt(false);
                setPendingUpdate(null);
                await applyUpdate(update);
              }}>Save & Restart</button>
            </div>
          </div>
        </div>
      )}

      {pendingDeleteID && (
        <div className="modal-overlay">
          <div className="modal">
            <h3>Delete Account</h3>
            <p>The account instance must be stopped before deletion.</p>
            <div className="modal-actions">
              <button onClick={() => setPendingDeleteID(null)}>Cancel</button>
              <button
                className="btn-danger"
                onClick={async () => {
                  const id = pendingDeleteID;
                  setPendingDeleteID(null);
                  await deleteAccount(id);
                }}
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading">Loading accounts...</div>
      ) : accounts.length === 0 ? (
        <div className="empty-state">
          <p>No accounts yet. Add your first account.</p>
        </div>
      ) : (
        <div className="accounts-list">
          {accounts.map(account => (
            <div key={account.id} className="account-card">
              <div className="account-info">
                <strong>{account.username || account.email}</strong>
                <span className="account-status">{account.status}</span>
              </div>
              <div className="account-meta">
                Group: {account.group || '-'} | Instance: {account.instance_name || account.instance_id || '-'} {account.instance_status ? `(${account.instance_status})` : ''}
              </div>
              <div className="account-meta">
                Proxy: {account.proxy_url || '-'} | Fingerprint: {account.fingerprint_country || 'US'} / {account.fingerprint_seed ? account.fingerprint_seed.slice(0, 8) + '...' : '-'}
              </div>
              {account.pending_restart && (
                <div className="warning-banner">Pending restart required</div>
              )}
              <div className="account-actions">
                <button className="btn-secondary" onClick={() => startEdit(account)}>Edit</button>
                {account.pending_restart && (
                  <button className="btn-primary" onClick={() => restartInstance(account)}>Restart</button>
                )}
                <button
                  className="btn-secondary"
                  onClick={() => setPendingDeleteID(account.id)}
                  disabled={!!account.instance_status && account.instance_status !== 'stopped'}
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
