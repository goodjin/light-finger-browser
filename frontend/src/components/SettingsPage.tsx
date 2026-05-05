import { useEffect, useState } from 'react';
import { ConnectRemote, DisconnectRemote, GetFingerprintCoverageReport } from '../wailsjs/go/main/App';
import { commands } from '../wailsjs/go/models';

interface RemoteConfig {
  host: string;
  port: number;
  binaryPath: string;
  connected: boolean;
}

export function SettingsPage() {
  const [chromePath, setChromePath] = useState('');
  const [remoteConfigs, setRemoteConfigs] = useState<RemoteConfig[]>([]);
  const [coverageReport, setCoverageReport] = useState<commands.FingerprintCoverageReport | null>(null);
  const [newRemote, setNewRemote] = useState<RemoteConfig>({
    host: '',
    port: 9222,
    binaryPath: '',
    connected: false,
  });
  const [connecting, setConnecting] = useState(false);
  const [disconnectingKey, setDisconnectingKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    loadCoverage();
  }, []);

  async function loadCoverage() {
    try {
      const report = await GetFingerprintCoverageReport();
      setCoverageReport(report || null);
    } catch (err) {
      setError(String(err));
    }
  }

  async function detectChrome() {
    setSuccess('Chrome detection would run here');
  }

  async function handleConnectRemote() {
    if (!newRemote.host) {
      setError('Host is required');
      return;
    }
    try {
      if (connecting) {
        return;
      }
      setConnecting(true);
      setError(null);
      await ConnectRemote(newRemote.host, newRemote.port, newRemote.binaryPath);
      setRemoteConfigs([...remoteConfigs, { ...newRemote, connected: true }]);
      setNewRemote({ host: '', port: 9222, binaryPath: '', connected: false });
      setSuccess('Connected to remote browser');
    } catch (err) {
      setError(String(err));
    } finally {
      setConnecting(false);
    }
  }

  async function handleDisconnect(config: RemoteConfig) {
    try {
      const key = `${config.host}:${config.port}`;
      if (disconnectingKey === key) {
        return;
      }
      setDisconnectingKey(key);
      setError(null);
      await DisconnectRemote(config.host, config.port);
      setRemoteConfigs(remoteConfigs.filter(r => r.host !== config.host));
      setSuccess('Disconnected');
    } catch (err) {
      setError(String(err));
    } finally {
      setDisconnectingKey(null);
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
          Local Chrome is available as a diagnostic fallback. Production instances use the self-built browser.
        </p>
        <div className="form-group">
          <label>Chrome Path</label>
          <div className="input-with-button">
                <input
                  type="text"
                  value={chromePath}
                  disabled={connecting}
                  onChange={e => setChromePath(e.target.value)}
                  placeholder="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
                />
                <button onClick={detectChrome} disabled={connecting}>Detect</button>
              </div>
            </div>
          </section>

      <section className="settings-section">
        <h3>Remote Browser</h3>
        <p className="section-desc">
          Connect to a remote browser host so browser processes run on another machine and are controlled via CDP.
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
                  disabled={disconnectingKey === `${config.host}:${config.port}`}
                >
                  {disconnectingKey === `${config.host}:${config.port}` ? 'Disconnecting...' : 'Disconnect'}
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
                  disabled={connecting}
                  onChange={e => setNewRemote({ ...newRemote, host: e.target.value })}
                  placeholder="192.168.1.100"
                />
            </div>
            <div className="form-group">
              <label>Port</label>
                <input
                  type="number"
                  value={newRemote.port}
                  disabled={connecting}
                  onChange={e => setNewRemote({ ...newRemote, port: parseInt(e.target.value) })}
                />
            </div>
          </div>
          <div className="form-group">
            <label>Browser Binary Path (optional)</label>
                <input
                  type="text"
                  value={newRemote.binaryPath}
                  disabled={connecting}
                  onChange={e => setNewRemote({ ...newRemote, binaryPath: e.target.value })}
                  placeholder="/path/to/browser"
                />
              </div>
          <button className="btn-primary" onClick={handleConnectRemote} disabled={connecting}>
            {connecting ? 'Connecting...' : 'Add Connection'}
          </button>
        </div>
      </section>

      <section className="settings-section">
        <h3>Fingerprint Runtime Coverage</h3>
        <p className="section-desc">
          Shows which generated fingerprint fields are actually injected into the running browser.
        </p>
        {coverageReport ? (
          <>
            <div className="coverage-summary">
              <strong>Active Engine:</strong> {coverageReport.active_engine}
            </div>
            <div className="coverage-list">
              {coverageReport.fields.map(field => (
                <div key={field.field} className="coverage-row">
                  <div className="coverage-field">{field.field}</div>
                  <div className="coverage-value">Local Chrome: {field.local_chrome}</div>
                  <div className="coverage-value">Self Built: {field.self_built}</div>
                  <div className="helper-text">{field.notes}</div>
                </div>
              ))}
            </div>
          </>
        ) : (
          <div className="text-muted">Coverage report unavailable.</div>
        )}
      </section>

      <section className="settings-section">
        <h3>About</h3>
        <p>fingerbrower v1.0.0</p>
        <p className="text-muted">Browser fingerprint management desktop application</p>
      </section>
    </div>
  );
}
