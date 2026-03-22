<script>
  import { onDestroy, createEventDispatcher } from 'svelte';
  import { GetWiFiDetail, GetWiFiPassword } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  const dispatch = createEventDispatcher();

  let data = null;
  let loading = true;
  let error = null;
  let interval = null;

  // Password reveal state: { [networkName]: { loading, password, error, shown } }
  let passwordState = {};

  async function fetchData() {
    try {
      const result = await GetWiFiDetail();
      data = result;
      error = null;
    } catch (e) {
      console.error('Failed to get WiFi detail:', e);
      error = 'Failed to load Wi-Fi data.';
    }
    loading = false;
  }

  function startPolling() {
    if (interval) clearInterval(interval);
    fetchData();
    interval = setInterval(fetchData, 5000);
  }

  function stopPolling() {
    if (interval) {
      clearInterval(interval);
      interval = null;
    }
  }

  $: if (visible) {
    startPolling();
  } else {
    stopPolling();
  }

  onDestroy(() => stopPolling());

  // --- Signal bar helpers ---
  // Returns 1–4 based on dBm value
  function signalBars(dbm) {
    if (dbm == null || dbm === 0) return 0;
    if (dbm > -50) return 4;
    if (dbm > -60) return 3;
    if (dbm > -70) return 2;
    return 1;
  }

  function signalLabel(dbm) {
    if (dbm == null || dbm === 0) return 'No signal';
    if (dbm > -50) return 'Excellent';
    if (dbm > -60) return 'Good';
    if (dbm > -70) return 'Fair';
    return 'Weak';
  }

  function signalColor(dbm) {
    if (dbm == null || dbm === 0) return 'var(--text-muted)';
    if (dbm > -50) return 'var(--green)';
    if (dbm > -60) return 'var(--blue)';
    if (dbm > -70) return 'var(--yellow)';
    return 'var(--red)';
  }

  // --- SNR helpers ---
  function snrValue(signal, noise) {
    if (signal == null || noise == null) return null;
    return signal - noise;
  }

  function snrLabel(snr) {
    if (snr == null) return '—';
    if (snr > 40) return 'Excellent';
    if (snr > 25) return 'Good';
    if (snr > 15) return 'Fair';
    return 'Poor';
  }

  function snrColor(snr) {
    if (snr == null) return 'var(--text-muted)';
    if (snr > 40) return 'var(--green)';
    if (snr > 25) return 'var(--blue)';
    if (snr > 15) return 'var(--yellow)';
    return 'var(--red)';
  }

  // Map SNR to a 0–100 percentage for the bar (cap at 50 dB = 100%)
  function snrPercent(snr) {
    if (snr == null) return 0;
    return Math.min(100, Math.max(0, (snr / 50) * 100));
  }

  // --- Nearby networks sorted by signal ---
  $: sortedNearby = data && data.nearbyNetworks
    ? [...data.nearbyNetworks].sort((a, b) => {
        // Parse leading dBm number from strings like "-65 dBm / -88 dBm"
        const parse = (s) => {
          if (!s) return -999;
          const m = s.match(/-?\d+/);
          return m ? parseInt(m[0], 10) : -999;
        };
        return parse(b.signal) - parse(a.signal);
      })
    : [];

  // Extract first dBm from a signal string like "-65 dBm / -88 dBm"
  function parseSignalDbm(s) {
    if (!s) return null;
    const m = s.match(/-?\d+/);
    return m ? parseInt(m[0], 10) : null;
  }

  // --- Password reveal ---
  async function togglePassword(name) {
    const current = passwordState[name] || {};

    if (current.shown) {
      // Hide
      passwordState = {
        ...passwordState,
        [name]: { ...current, shown: false },
      };
      return;
    }

    if (current.password) {
      // Already fetched, just show
      passwordState = {
        ...passwordState,
        [name]: { ...current, shown: true },
      };
      return;
    }

    // Fetch
    passwordState = {
      ...passwordState,
      [name]: { loading: true, password: null, error: null, shown: false },
    };

    try {
      const pw = await GetWiFiPassword(name);
      passwordState = {
        ...passwordState,
        [name]: { loading: false, password: pw, error: null, shown: true },
      };
    } catch (e) {
      passwordState = {
        ...passwordState,
        [name]: {
          loading: false,
          password: null,
          error: 'Could not retrieve password.',
          shown: false,
        },
      };
    }
  }
</script>

<div class="wifi-detail">
  <!-- Header -->
  <div class="detail-header">
    <button class="back-btn" on:click={() => dispatch('back')} title="Go back">
      &#8592; Back
    </button>
    <div class="header-text">
      <div class="title-row">
        <h1 class="page-title">Wi-Fi</h1>
        {#if data}
          <span
            class="power-badge"
            class:on={data.powerOn}
            class:off={!data.powerOn}
          >{data.powerOn ? 'On' : 'Off'}</span>
        {/if}
      </div>
      {#if data && data.connected && data.ssid}
        <p class="page-subtitle">Connected to {data.ssid}</p>
      {:else if data}
        <p class="page-subtitle not-connected">Not connected</p>
      {/if}
    </div>
  </div>

  {#if loading && !data}
    <div class="loading-state">Loading Wi-Fi data&hellip;</div>
  {:else if error && !data}
    <div class="error-state">{error}</div>
  {:else if data}
    <div class="content">

      <!-- Current Connection card -->
      {#if data.connected && data.ssid}
        <div class="card connection-card">
          <div class="card-label">Current Connection</div>

          <div class="connection-main">
            <!-- Left: SSID + signal bars -->
            <div class="connection-left">
              <div class="wifi-icon-wrap">
                <!-- SVG WiFi bars icon -->
                <svg
                  width="40"
                  height="32"
                  viewBox="0 0 40 32"
                  fill="none"
                  aria-label="Wi-Fi signal: {signalLabel(data.signalStrength)}"
                >
                  {#each [1, 2, 3, 4] as bar}
                    {@const bars = signalBars(data.signalStrength)}
                    {@const active = bar <= bars}
                    <!-- Arc arcs: bar 1 = outermost, 4 = innermost dot -->
                    {#if bar === 4}
                      <!-- Dot -->
                      <circle
                        cx="20"
                        cy="29"
                        r="3"
                        fill={active ? signalColor(data.signalStrength) : 'var(--bg-tertiary)'}
                      />
                    {:else}
                      <!-- Arcs: bar 3 = small, 2 = medium, 1 = large -->
                      {@const arcBar = 4 - bar}
                      {@const r = 6 + (arcBar - 1) * 7}
                      {@const sweep = 0.6}
                      {@const cx = 20}
                      {@const cy = 30}
                      {@const x1 = cx - r * Math.sin(sweep)}
                      {@const y1 = cy - r * Math.cos(sweep)}
                      {@const x2 = cx + r * Math.sin(sweep)}
                      {@const y2 = cy - r * Math.cos(sweep)}
                      <path
                        d="M {x1.toFixed(2)} {y1.toFixed(2)} A {r} {r} 0 0 1 {x2.toFixed(2)} {y2.toFixed(2)}"
                        stroke={active ? signalColor(data.signalStrength) : 'var(--bg-tertiary)'}
                        stroke-width="3.5"
                        stroke-linecap="round"
                        fill="none"
                      />
                    {/if}
                  {/each}
                </svg>
              </div>
              <div class="connection-ssid-wrap">
                <div class="connection-ssid">{data.ssid}</div>
                <div class="signal-quality-text" style="color: {signalColor(data.signalStrength)}">
                  {signalLabel(data.signalStrength)} &mdash; {data.signalStrength} dBm
                </div>
              </div>
            </div>

            <!-- Right: security badge -->
            {#if data.security}
              <span class="security-badge">{data.security}</span>
            {/if}
          </div>

          <!-- Connection details row -->
          <div class="connection-details">
            {#if data.channel}
              <div class="conn-detail-item">
                <span class="conn-detail-label">Channel</span>
                <span class="conn-detail-value">{data.channel}</span>
              </div>
            {/if}
            {#if data.band}
              <div class="conn-detail-item">
                <span class="conn-detail-label">Band</span>
                <span class="conn-detail-value">{data.band}</span>
              </div>
            {/if}
            {#if data.phyMode}
              <div class="conn-detail-item">
                <span class="conn-detail-label">PHY Mode</span>
                <span class="conn-detail-value mono">{data.phyMode}</span>
              </div>
            {/if}
            {#if data.transmitRate}
              <div class="conn-detail-item">
                <span class="conn-detail-label">Tx Rate</span>
                <span class="conn-detail-value">{data.transmitRate} Mbps</span>
              </div>
            {/if}
          </div>
        </div>
      {:else}
        <div class="card not-connected-card">
          <div class="not-connected-inner">
            <svg width="36" height="36" viewBox="0 0 36 36" fill="none" aria-hidden="true">
              <circle cx="18" cy="18" r="17" stroke="var(--border)" stroke-width="2"/>
              <line x1="11" y1="11" x2="25" y2="25" stroke="var(--text-muted)" stroke-width="2.5" stroke-linecap="round"/>
              <line x1="25" y1="11" x2="11" y2="25" stroke="var(--text-muted)" stroke-width="2.5" stroke-linecap="round"/>
            </svg>
            <div>
              <div class="nc-title">Not Connected</div>
              <div class="nc-sub">Wi-Fi is {data.powerOn ? 'on but not connected' : 'turned off'}</div>
            </div>
          </div>
        </div>
      {/if}

      <!-- Signal Quality section -->
      {#if data.connected && data.signalStrength != null && data.noiseLevel != null}
        {@const snr = snrValue(data.signalStrength, data.noiseLevel)}
        <div class="card signal-card">
          <div class="card-label">Signal Quality</div>
          <div class="signal-grid">
            <!-- Signal -->
            <div class="signal-metric">
              <div class="signal-metric-label">Signal Strength</div>
              <div class="signal-metric-value" style="color: {signalColor(data.signalStrength)}">
                {data.signalStrength} dBm
              </div>
            </div>
            <!-- Noise -->
            <div class="signal-metric">
              <div class="signal-metric-label">Noise Level</div>
              <div class="signal-metric-value">{data.noiseLevel} dBm</div>
            </div>
            <!-- SNR -->
            <div class="signal-metric snr-metric">
              <div class="signal-metric-label">
                SNR
                <span class="snr-badge" style="color: {snrColor(snr)}; background: color-mix(in srgb, {snrColor(snr)} 14%, transparent);">
                  {snrLabel(snr)}
                </span>
              </div>
              <div class="signal-metric-value" style="color: {snrColor(snr)}">
                {snr} dB
              </div>
              <div class="snr-bar-track">
                <div
                  class="snr-bar-fill"
                  style="width: {snrPercent(snr)}%; background: {snrColor(snr)};"
                ></div>
              </div>
            </div>
          </div>
        </div>
      {/if}

      <!-- Hardware Info grid -->
      <div class="section-label">Hardware Info</div>
      <div class="hw-grid">
        {#if data.macAddress}
          <div class="hw-card">
            <div class="hw-icon-wrap accent">
              <!-- Network/MAC icon -->
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                <rect x="1" y="5" width="14" height="6" rx="2" stroke="currentColor" stroke-width="1.5"/>
                <circle cx="4.5" cy="8" r="1" fill="currentColor"/>
                <circle cx="8" cy="8" r="1" fill="currentColor"/>
                <circle cx="11.5" cy="8" r="1" fill="currentColor"/>
              </svg>
            </div>
            <div class="hw-body">
              <div class="hw-name">MAC Address</div>
              <div class="hw-val mono">{data.macAddress}</div>
            </div>
          </div>
        {/if}

        {#if data.cardType}
          <div class="hw-card">
            <div class="hw-icon-wrap blue">
              <!-- Chip icon -->
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                <rect x="4" y="4" width="8" height="8" rx="1.5" stroke="currentColor" stroke-width="1.5"/>
                <line x1="6" y1="1.5" x2="6" y2="4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="10" y1="1.5" x2="10" y2="4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="6" y1="12" x2="6" y2="14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="10" y1="12" x2="10" y2="14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="1.5" y1="6" x2="4" y2="6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="1.5" y1="10" x2="4" y2="10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="12" y1="6" x2="14.5" y2="6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="12" y1="10" x2="14.5" y2="10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
            </div>
            <div class="hw-body">
              <div class="hw-name">Card Type</div>
              <div class="hw-val mono small">{data.cardType}</div>
            </div>
          </div>
        {/if}

        {#if data.countryCode}
          <div class="hw-card">
            <div class="hw-icon-wrap green">
              <!-- Globe icon -->
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                <circle cx="8" cy="8" r="6.5" stroke="currentColor" stroke-width="1.5"/>
                <path d="M8 1.5 C6 4 6 12 8 14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <path d="M8 1.5 C10 4 10 12 8 14.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="1.5" y1="8" x2="14.5" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
            </div>
            <div class="hw-body">
              <div class="hw-name">Country Code</div>
              <div class="hw-val">{data.countryCode}</div>
            </div>
          </div>
        {/if}

        {#if data.supportedPHY}
          <div class="hw-card">
            <div class="hw-icon-wrap yellow">
              <!-- Signal icon -->
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                <path d="M1.5 12 C3.5 8 7 6 8 6 C9 6 12.5 8 14.5 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <path d="M4 13.5 C5.2 11 6.6 10 8 10 C9.4 10 10.8 11 12 13.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <circle cx="8" cy="14.5" r="1.2" fill="currentColor"/>
              </svg>
            </div>
            <div class="hw-body">
              <div class="hw-name">Supported PHY</div>
              <div class="hw-val mono">{data.supportedPHY}</div>
            </div>
          </div>
        {/if}

        {#if data.airdrop}
          <div class="hw-card">
            <div class="hw-icon-wrap accent">
              <!-- Share icon -->
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                <circle cx="12.5" cy="3.5" r="2" stroke="currentColor" stroke-width="1.5"/>
                <circle cx="12.5" cy="12.5" r="2" stroke="currentColor" stroke-width="1.5"/>
                <circle cx="3.5" cy="8" r="2" stroke="currentColor" stroke-width="1.5"/>
                <line x1="5.4" y1="7" x2="10.7" y2="4.3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <line x1="5.4" y1="9" x2="10.7" y2="11.7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
            </div>
            <div class="hw-body">
              <div class="hw-name">AirDrop</div>
              <div class="hw-val">{data.airdrop}</div>
            </div>
          </div>
        {/if}

        {#if data.autoUnlock}
          <div class="hw-card">
            <div class="hw-icon-wrap green">
              <!-- Lock icon -->
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                <rect x="3" y="7" width="10" height="8" rx="2" stroke="currentColor" stroke-width="1.5"/>
                <path d="M5 7V5a3 3 0 0 1 6 0v2" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <circle cx="8" cy="11" r="1.2" fill="currentColor"/>
              </svg>
            </div>
            <div class="hw-body">
              <div class="hw-name">Auto Unlock</div>
              <div class="hw-val">{data.autoUnlock}</div>
            </div>
          </div>
        {/if}
      </div>

      <!-- Saved Networks -->
      {#if data.savedNetworks && data.savedNetworks.length > 0}
        <div class="section-label">
          Saved Networks
          <span class="section-count">{data.savedNetworks.length}</span>
        </div>
        <div class="card networks-card">
          <div class="networks-list saved-list">
            {#each data.savedNetworks as net}
              {@const isActive = data.connected && data.ssid === net.name}
              {@const ps = passwordState[net.name] || {}}
              <div class="network-row" class:active-network={isActive}>
                <div class="network-row-left">
                  <!-- Small wifi icon -->
                  <svg width="18" height="14" viewBox="0 0 18 14" fill="none" aria-hidden="true">
                    <circle cx="9" cy="13" r="1.4" fill={isActive ? 'var(--accent)' : 'var(--text-muted)'}/>
                    <path d="M5.5 9.8 A4.8 4.8 0 0 1 12.5 9.8" stroke={isActive ? 'var(--accent)' : 'var(--text-muted)'} stroke-width="2" stroke-linecap="round" fill="none"/>
                    <path d="M2.5 6.8 A8 8 0 0 1 15.5 6.8" stroke={isActive ? 'var(--accent)' : 'var(--border)'} stroke-width="2" stroke-linecap="round" fill="none"/>
                    <path d="M0 3.8 A11.5 11.5 0 0 1 18 3.8" stroke={isActive ? 'var(--accent)' : 'var(--border)'} stroke-width="2" stroke-linecap="round" fill="none"/>
                  </svg>
                  <div class="network-name-wrap">
                    <span class="network-name" class:active-name={isActive}>{net.name}</span>
                    {#if isActive}
                      <span class="connected-dot">&#9679; Connected</span>
                    {/if}
                  </div>
                </div>
                <div class="network-row-right">
                  {#if ps.shown && ps.password}
                    <span class="password-reveal mono">{ps.password}</span>
                  {:else if ps.error}
                    <span class="password-error">{ps.error}</span>
                  {/if}
                  <button
                    class="pw-btn"
                    on:click={() => togglePassword(net.name)}
                    title={ps.shown ? 'Hide password' : 'Show password'}
                    disabled={ps.loading}
                  >
                    {#if ps.loading}
                      <span class="pw-btn-spinner">&#8987;</span>
                    {:else if ps.shown}
                      <!-- Eye-off SVG -->
                      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
                        <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
                        <line x1="1" y1="1" x2="23" y2="23"/>
                      </svg>
                    {:else}
                      <!-- Eye SVG -->
                      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                        <circle cx="12" cy="12" r="3"/>
                      </svg>
                    {/if}
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Nearby Networks -->
      {#if sortedNearby && sortedNearby.length > 0}
        <div class="section-label">
          Nearby Networks
          <span class="section-count">{sortedNearby.length}</span>
        </div>
        <div class="card networks-card">
          <div class="networks-list nearby-list">
            <!-- Header row -->
            <div class="nearby-header">
              <span>Network</span>
              <span>Signal</span>
              <span>Channel</span>
              <span>Security</span>
              <span>PHY</span>
            </div>
            {#each sortedNearby as net}
              {@const dbm = parseSignalDbm(net.signal)}
              {@const bars = signalBars(dbm)}
              {@const isActive = data.connected && data.ssid === net.name}
              <div class="nearby-row" class:active-network={isActive}>
                <!-- Name + bars -->
                <div class="nearby-name-cell">
                  <!-- Inline WiFi bar icon -->
                  <span class="nearby-bars" title="{signalLabel(dbm)} ({dbm} dBm)">
                    {#each [1, 2, 3, 4] as b}
                      <span
                        class="nearby-bar"
                        style="
                          height: {4 + (b - 1) * 3}px;
                          background: {b <= bars ? signalColor(dbm) : 'var(--bg-tertiary)'};
                        "
                      ></span>
                    {/each}
                  </span>
                  <span class="nearby-name" class:active-name={isActive}>{net.name}</span>
                </div>
                <span class="nearby-signal" style="color: {signalColor(dbm)}">{dbm != null ? dbm + ' dBm' : net.signal || '—'}</span>
                <span class="nearby-channel">{net.channel || '—'}</span>
                <span class="nearby-security">{net.security || '—'}</span>
                <span class="nearby-phy mono">{net.phyMode || '—'}</span>
              </div>
            {/each}
          </div>
        </div>
      {/if}

    </div>
  {/if}
</div>

<style>
  /* ── Layout ── */
  .wifi-detail {
    padding: 0 32px 32px;
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  /* ── Header ── */
  .detail-header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 24px;
  }

  .back-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 7px 14px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    font-size: 13px;
    font-weight: 500;
    transition: all var(--transition);
    flex-shrink: 0;
  }

  .back-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
    border-color: var(--accent);
  }

  .header-text {
    flex: 1;
    min-width: 0;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .page-title {
    font-size: 24px;
    font-weight: 700;
    letter-spacing: -0.5px;
    line-height: 1;
  }

  .power-badge {
    display: inline-flex;
    align-items: center;
    padding: 2px 9px;
    border-radius: 20px;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.3px;
    text-transform: uppercase;
  }

  .power-badge.on {
    background: var(--green-dim);
    color: var(--green);
  }

  .power-badge.off {
    background: var(--red-dim);
    color: var(--red);
  }

  .page-subtitle {
    color: var(--text-secondary);
    font-size: 13px;
    margin-top: 4px;
  }

  .page-subtitle.not-connected {
    color: var(--text-muted);
  }

  /* ── States ── */
  .loading-state,
  .error-state {
    color: var(--text-secondary);
    padding: 60px 0;
    text-align: center;
  }

  .error-state {
    color: var(--red);
  }

  /* ── Content ── */
  .content {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  /* ── Cards ── */
  .card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 24px;
  }

  .card-label {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.6px;
    text-transform: uppercase;
    color: var(--text-muted);
    margin-bottom: 18px;
  }

  /* ── Section labels ── */
  .section-label {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.6px;
    text-transform: uppercase;
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
  }

  .section-count {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    border-radius: 20px;
    padding: 1px 7px;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0;
    text-transform: none;
  }

  /* ── Connection card ── */
  .connection-card {
    border-color: var(--accent);
    background: linear-gradient(135deg, var(--bg-card) 0%, color-mix(in srgb, var(--accent) 4%, var(--bg-card)) 100%);
  }

  .connection-main {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 20px;
  }

  .connection-left {
    display: flex;
    align-items: center;
    gap: 16px;
    min-width: 0;
  }

  .wifi-icon-wrap {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 56px;
    height: 56px;
    background: var(--accent-dim);
    border-radius: var(--radius);
  }

  .connection-ssid-wrap {
    min-width: 0;
  }

  .connection-ssid {
    font-size: 20px;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.3px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .signal-quality-text {
    font-size: 12px;
    font-weight: 500;
    margin-top: 4px;
  }

  .security-badge {
    flex-shrink: 0;
    padding: 4px 10px;
    background: var(--accent-dim);
    color: var(--accent);
    border: 1px solid color-mix(in srgb, var(--accent) 30%, transparent);
    border-radius: 20px;
    font-size: 11px;
    font-weight: 600;
    white-space: nowrap;
  }

  .connection-details {
    display: flex;
    flex-wrap: wrap;
    gap: 0;
    border-top: 1px solid var(--border);
    padding-top: 16px;
  }

  .conn-detail-item {
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 8px 20px;
    border-right: 1px solid var(--border);
  }

  .conn-detail-item:first-child {
    padding-left: 0;
  }

  .conn-detail-item:last-child {
    border-right: none;
  }

  .conn-detail-label {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
  }

  .conn-detail-value {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .conn-detail-value.mono {
    font-family: 'SF Mono', 'Menlo', 'Consolas', monospace;
  }

  /* ── Not connected card ── */
  .not-connected-card {
    border-style: dashed;
  }

  .not-connected-inner {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .nc-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .nc-sub {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 3px;
  }

  /* ── Signal quality card ── */
  .signal-grid {
    display: grid;
    grid-template-columns: 1fr 1fr 2fr;
    gap: 24px;
    align-items: start;
  }

  .signal-metric-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
    margin-bottom: 6px;
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }

  .signal-metric-value {
    font-size: 22px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.5px;
  }

  .snr-metric {
    grid-column: span 1;
  }

  .snr-badge {
    display: inline-flex;
    align-items: center;
    padding: 1px 7px;
    border-radius: 20px;
    font-size: 10px;
    font-weight: 600;
    text-transform: none;
    letter-spacing: 0;
  }

  .snr-bar-track {
    height: 6px;
    background: var(--bg-tertiary);
    border-radius: 3px;
    overflow: hidden;
    margin-top: 10px;
  }

  .snr-bar-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.6s ease;
  }

  /* ── Hardware grid ── */
  .hw-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
  }

  .hw-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 18px 20px;
    display: flex;
    align-items: flex-start;
    gap: 14px;
    transition: border-color var(--transition);
  }

  .hw-card:hover {
    border-color: var(--accent);
  }

  .hw-icon-wrap {
    width: 34px;
    height: 34px;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .hw-icon-wrap.accent { background: var(--accent-dim); color: var(--accent); }
  .hw-icon-wrap.blue   { background: var(--blue-dim);   color: var(--blue);   }
  .hw-icon-wrap.green  { background: var(--green-dim);  color: var(--green);  }
  .hw-icon-wrap.yellow { background: var(--yellow-dim); color: var(--yellow); }

  .hw-body {
    flex: 1;
    min-width: 0;
  }

  .hw-name {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
    margin-bottom: 4px;
  }

  .hw-val {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .hw-val.mono {
    font-family: 'SF Mono', 'Menlo', 'Consolas', monospace;
    font-size: 12px;
    font-weight: 500;
  }

  .hw-val.small {
    font-size: 11px;
  }

  /* ── Networks card ── */
  .networks-card {
    padding: 0;
    overflow: hidden;
  }

  .networks-list {
    display: flex;
    flex-direction: column;
  }

  /* Saved networks */
  .network-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 20px;
    border-bottom: 1px solid var(--border);
    transition: background var(--transition);
    gap: 12px;
  }

  .network-row:last-child {
    border-bottom: none;
  }

  .network-row:hover {
    background: var(--bg-hover);
  }

  .network-row.active-network {
    background: var(--accent-dim);
  }

  .network-row-left {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
    flex: 1;
  }

  .network-name-wrap {
    min-width: 0;
  }

  .network-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
    display: block;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .network-name.active-name {
    color: var(--accent);
    font-weight: 600;
  }

  .connected-dot {
    font-size: 10px;
    color: var(--accent);
    font-weight: 500;
    margin-top: 2px;
    display: block;
  }

  .network-row-right {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  .password-reveal {
    font-family: 'SF Mono', 'Menlo', 'Consolas', monospace;
    font-size: 12px;
    color: var(--green);
    background: var(--green-dim);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    user-select: text;
  }

  .password-error {
    font-size: 12px;
    color: var(--red);
  }

  .pw-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    transition: all var(--transition);
    flex-shrink: 0;
  }

  .pw-btn:hover:not(:disabled) {
    background: var(--bg-hover);
    color: var(--text-primary);
    border-color: var(--accent);
  }

  .pw-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .pw-btn-spinner {
    font-size: 13px;
    animation: spin 1s linear infinite;
    display: inline-block;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to   { transform: rotate(360deg); }
  }

  /* Nearby networks */
  .nearby-header {
    display: grid;
    grid-template-columns: 2fr 1fr 1.5fr 1.5fr 1fr;
    gap: 12px;
    padding: 10px 20px;
    border-bottom: 1px solid var(--border);
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
  }

  .nearby-row {
    display: grid;
    grid-template-columns: 2fr 1fr 1.5fr 1.5fr 1fr;
    gap: 12px;
    padding: 11px 20px;
    border-bottom: 1px solid var(--border);
    align-items: center;
    transition: background var(--transition);
  }

  .nearby-row:last-child {
    border-bottom: none;
  }

  .nearby-row:hover {
    background: var(--bg-hover);
  }

  .nearby-row.active-network {
    background: var(--accent-dim);
  }

  .nearby-name-cell {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .nearby-bars {
    display: flex;
    align-items: flex-end;
    gap: 2px;
    flex-shrink: 0;
    height: 14px;
  }

  .nearby-bar {
    width: 3px;
    border-radius: 1.5px;
    display: block;
  }

  .nearby-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .nearby-name.active-name {
    color: var(--accent);
    font-weight: 600;
  }

  .nearby-signal {
    font-size: 12px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .nearby-channel,
  .nearby-security {
    font-size: 12px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .nearby-phy {
    font-family: 'SF Mono', 'Menlo', 'Consolas', monospace;
    font-size: 11px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mono {
    font-family: 'SF Mono', 'Menlo', 'Consolas', monospace;
  }

  /* ── Responsive ── */
  @media (max-width: 900px) {
    .signal-grid {
      grid-template-columns: 1fr 1fr;
    }

    .snr-metric {
      grid-column: span 2;
    }

    .hw-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .nearby-header,
    .nearby-row {
      grid-template-columns: 2fr 1fr 1fr 1fr;
    }

    .nearby-header > span:nth-child(4),
    .nearby-row > .nearby-security {
      display: none;
    }
  }

  @media (max-width: 640px) {
    .wifi-detail {
      padding: 0 16px 24px;
    }

    .hw-grid {
      grid-template-columns: 1fr;
    }

    .signal-grid {
      grid-template-columns: 1fr;
    }

    .snr-metric {
      grid-column: span 1;
    }

    .connection-main {
      flex-direction: column;
      align-items: flex-start;
    }

    .nearby-header,
    .nearby-row {
      grid-template-columns: 2fr 1fr 1fr;
    }

    .nearby-header > span:nth-child(3),
    .nearby-row > .nearby-channel {
      display: none;
    }
  }
</style>
