import { useState, useEffect } from 'react';

interface Account {
  id: string;
  username: string;
  email: string;
  status: string;
  group: string;
  instance_id: string;
  created_at: string;
}

export function AccountsPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form state
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');

  useEffect(() => {
    loadAccounts();
  }, []);

  async function loadAccounts() {
    setLoading(true);
    try {
      // TODO: Implement account listing
      setAccounts([]);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function createAccount() {
    if (!email) {
      setError('Email is required');
      return;
    }
    try {
      // TODO: Implement account creation
      setShowCreate(false);
      setUsername('');
      setEmail('');
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
              <label>Email</label>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="account@example.com"
                required
              />
            </div>
            <div className="modal-actions">
              <button onClick={() => setShowCreate(false)}>Cancel</button>
              <button className="btn-primary" onClick={createAccount}>Add</button>
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
                Group: {account.group || '-'} | Instance: {account.instance_id || '-'}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
