import { useState } from 'react';
import { ConnectRemote, DisconnectRemote } from '../wailsjs/go/main';

interface RemoteConfig {
  host: string;
  port: number;
  binaryPath: string;
  connected: boolean;
}

export function SettingsPage() {
  const [chromePath, setChromePath] = useState('');
  const [remoteConfigs, setRemoteConfigs] = useState<RemoteConfig[]>([]);
  const [newRemote, setNewRemote] = useState<RemoteConfig>({
    host: '',
    port: 9222,
    binaryPath: '',
    connected: false,
  });
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function detectChrome() {
    setSuccess('Chrome detection would run here');
  }

  async function handleConnectRemote() {
    if (!newRemote.host) {
      setError('Host is required');
      return;
    }
    try {
      await ConnectRemote(newRemote.host, newRemote.port, newRemote.binaryPath);
      setRemoteConfigs([...remoteConfigs, { ...newRemote, connected: true }]);
      setNewRemote({ host: '', port: 9222, binaryPath: '', connected: false });
      setSuccess('Connected to remote CloakBrowser');
    } catch (err) {
      setError(String(err));
    }
  }

  async function handleDisconnect(config: RemoteConfig) {
    try {
      await DisconnectRemote(config.host, config.port);
      setRemoteConfigs(remoteConfigs.filter(r => r.host !== config.host));
      setSuccess('Disconnected');
    } catch (err) {
      setError(String(err));
    }
  }

  return (
    <div className="settings-page">
      <h2>Settings</h2>

      {error && <div className="error-banner">{error}</div>}
      {success && <div className="success-banner">{success}</div>}

      <section className="settings-section">
        <h3>Local Chrome</h3>
        <p className="section-desc">
          Configure the local Chrome browser for fingerprint injection.
        </p>
        <div className="form-group">
          <label>Chrome Path</label>
          <div className="input-with-button">
            <input
              type="text"
              value={chromePath}
              onChange={e => setChromePath(e.target.value)}
              placeholder="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
            />
            <button onClick={detectChrome}>Detect</button>
          </div>
        </div>
      </section>

      <section className="settings-section">
        <h3>Remote CloakBrowser</h3>
        <p className="section-desc">
          Connect to remote CloakBrowser instances for distributed browsing.
        </p>

        <div className="remote-list">
          {remoteConfigs.map((config, idx) => (
            <div key={idx} className="remote-item">
              <div className="remote-info">
                <strong>{config.host}:{config.port}</strong>
                <span className={config.connected ? 'status-connected' : 'status-disconnected'}>
                  {config.connected ? 'Connected' : 'Disconnected'}
                </span>
              </div>
              {config.connected && (
                <button
                  className="btn-small btn-danger"
                  onClick={() => handleDisconnect(config)}
                >
                  Disconnect
                </button>
              )}
            </div>
          ))}
        </div>

        <div className="add-remote-form">
          <h4>Add Remote Connection</h4>
          <div className="form-row">
            <div className="form-group">
              <label>Host</label>
              <input
                type="text"
                value={newRemote.host}
                onChange={e => setNewRemote({ ...newRemote, host: e.target.value })}
                placeholder="192.168.1.100"
              />
            </div>
            <div className="form-group">
              <label>Port</label>
              <input
                type="number"
                value={newRemote.port}
                onChange={e => setNewRemote({ ...newRemote, port: parseInt(e.target.value) })}
              />
            </div>
          </div>
          <div className="form-group">
            <label>CloakBrowser Binary Path (optional)</label>
            <input
              type="text"
              value={newRemote.binaryPath}
              onChange={e => setNewRemote({ ...newRemote, binaryPath: e.target.value })}
              placeholder="/path/to/cloakbrowser"
            />
          </div>
          <button className="btn-primary" onClick={handleConnectRemote}>
            Add Connection
          </button>
        </div>
      </section>

      <section className="settings-section">
        <h3>About</h3>
        <p>fingerbrower v1.0.0</p>
        <p className="text-muted">Browser fingerprint management desktop application</p>
      </section>
    </div>
  );
}
