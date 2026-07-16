<script>
  import { settings, setError, setNotice, clearBanners } from '../../lib/stores.js';
  import { putJSON } from '../../lib/api.js';
  import { t } from '../../lib/preferences.js';
  let form = { defaultMarket: 'skillhub.cn', defaultInstallScope: 'project', officialHandles: [], markets: [] };
  let last = '';
  $: if ($settings !== last) { last = $settings; load($settings); }
  function load(raw) { try { const cfg = JSON.parse(raw || '{}'); const hub = cfg.skillHub || {}; form = { defaultMarket: hub.defaultMarket || 'skillhub.cn', defaultInstallScope: hub.defaultInstallScope || 'project', officialHandles: hub.officialHandles || [], markets: hub.markets || [] }; } catch (e) { setError(e); } }
  function addMarket() { form.markets = [...form.markets, { id: '', name: '', siteURL: '', apiURL: '', enabled: true, apiToken: '' }]; }
  function removeMarket(i) { form.markets = form.markets.filter((_, n) => n !== i); }
  async function save() { clearBanners(); try { const cfg = JSON.parse($settings || '{}'); cfg.skillHub = form; const saved = await putJSON('/api/settings', cfg); settings.set(JSON.stringify(saved, null, 2)); setNotice('SkillHub settings saved.'); } catch (e) { setError(e); } }
</script>
<section class="settings-card">
  <div class="settings-card-head"><div><h3>SkillHub</h3><span class="hint">Markets, defaults, Official handles and registry tokens.</span></div><button class="primary" on:click={save}>Save</button></div>
  <div class="settings-form-grid">
    <label><span>Default market</span><input bind:value={form.defaultMarket} /></label>
    <label><span>Install scope</span><select bind:value={form.defaultInstallScope}><option value="project">Project</option><option value="global">Global</option></select></label>
    <label class="full"><span>Official handles (comma separated)</span><input value={form.officialHandles.join(', ')} on:change={(e) => form.officialHandles = e.currentTarget.value.split(',').map((v) => v.trim()).filter(Boolean)} /></label>
  </div>
  <div class="settings-card-head"><h4>Markets</h4><button type="button" on:click={addMarket}>Add market</button></div>
  {#each form.markets as market, i}
    <div class="settings-form-grid market-editor">
      <label><span>ID</span><input bind:value={market.id} placeholder="skillhub.cn" /></label>
      <label><span>Name</span><input bind:value={market.name} /></label>
      <label><span>Site URL</span><input bind:value={market.siteURL} /></label>
      <label><span>API URL</span><input bind:value={market.apiURL} /></label>
      <label><span>Bearer token</span><input type="password" bind:value={market.apiToken} /></label>
      <label class="checkbox"><input type="checkbox" bind:checked={market.enabled} /> Enabled</label>
      <button type="button" on:click={() => removeMarket(i)}>Remove</button>
    </div>
  {/each}
</section>
