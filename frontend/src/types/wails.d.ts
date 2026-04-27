// Type definitions for Wails generated bindings

export interface ScreenConfig {
  width: number;
  height: number;
  pixel_ratio: number;
}

export interface WebGLConfig {
  renderer: string;
  vendor: string;
  extensions: string[];
}

export interface CanvasConfig {
  hash: string;
}

export interface AudioConfig {
  hash: string;
}

export interface HardwareConfig {
  cpu_cores: number;
  memory_gb: number;
  gpu_vendor: string;
  gpu_model: string;
  gpu_renderer: string;
}

export interface NetworkConfig {
  connection_type: string;
  downlink: number;
  rtt: number;
}

export interface Fingerprint {
  seed: string;
  user_agent: string;
  platform: string;
  screen: ScreenConfig;
  timezone: string;
  locale: string;
  canvas: CanvasConfig;
  webgl: WebGLConfig;
  audio: AudioConfig;
  hardware: HardwareConfig;
  network: NetworkConfig;
}

export interface ProxyConfig {
  id: string;
  url: string;
  type: string;
}

export interface InstanceConfig {
  fingerprint: Fingerprint | null;
  proxy?: ProxyConfig | null;
  account_id: string;
  group: string;
  headless: boolean;
}

export interface BrowserInstance {
  id: string;
  status: 'pending' | 'starting' | 'running' | 'stopping' | 'stopped' | 'error';
  fingerprint: Fingerprint | null;
  proxy_id: string;
  account_id: string;
  cdp_endpoint: string;
  pid: number;
  port: number;
  user_data_dir: string;
  group: string;
  started_at: string;
  last_active_at: string;
  created_at: string;
}

export interface InstanceFilter {
  status?: 'pending' | 'starting' | 'running' | 'stopping' | 'stopped' | 'error';
  group?: string;
  proxy_id?: string;
  account_id?: string;
}

export interface CDPTarget {
  id: string;
  type: string;
  title: string;
  url: string;
  webSocketDebuggerUrl: string;
}

export function CreateInstance(cfg: InstanceConfig): Promise<BrowserInstance>;
export function DestroyInstance(id: string): Promise<void>;
export function GetInstance(id: string): Promise<BrowserInstance>;
export function ListInstances(filter: InstanceFilter | null): Promise<BrowserInstance[]>;
export function GenerateFingerprint(seed: string, country: string): Promise<Fingerprint>;
export function GenerateRandomFingerprint(country: string): Promise<Fingerprint>;
export function ValidateFingerprint(fp: Fingerprint): Promise<void>;
export function ConnectRemote(host: string, port: number, binaryPath: string): Promise<void>;
export function DisconnectRemote(host: string, port: number): Promise<void>;
export function ListRemoteTargets(host: string, port: number): Promise<CDPTarget[]>;
export function GetRemoteCDPEndpoint(host: string, port: number): Promise<string>;
export function Greet(name: string): Promise<string>;